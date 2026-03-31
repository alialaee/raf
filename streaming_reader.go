package raf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var ErrKeyAlreadyPassed = errors.New("raf: key position already passed in stream")

// StreamBlock reads a RAF block sequentially from an io.Reader.
// After calling ReadHeader, values are read in order using Next,
// or skipped using Skip and SkipTo.
//
// The Value returned by Next reuses an internal buffer and is only
// valid until the next call to Next.
type StreamBlock struct {
	r          io.Reader
	headerRead bool
	isNull     bool
	pairsCount int
	valueTypes []byte
	keys       [][]byte
	valSizes   []int //  byte size of each value

	nextValIndex int

	headerBuf []byte
	valueBuf  []byte

	// innerReader is set when the caller obtained an inner StreamBlock via
	// NextMap but has not yet consumed it fully. Next, Skip, and NextBlock
	// auto-drain it before advancing.
	innerReader      *io.LimitedReader
	innerStreamBlock *StreamBlock
	innerStreamArray *StreamArray
}

// NewStreamBlock creates a StreamBlock that reads from r.
func NewStreamBlock(r io.Reader) *StreamBlock {
	return &StreamBlock{r: r}
}

// Reset resets the StreamBlock to read from a new reader.
func (sb *StreamBlock) Reset(r io.Reader) {
	sb.r = r
	sb.headerRead = false
	sb.isNull = false
	sb.pairsCount = 0
	sb.valueTypes = nil
	sb.nextValIndex = 0
	sb.keys = sb.keys[:0]
	sb.valSizes = sb.valSizes[:0]
	sb.headerBuf = sb.headerBuf[:0]
	sb.valueBuf = sb.valueBuf[:0]
}

// ReadHeader reads the block header and returns the sorted keys.
// Must be called before Next, Skip, SkipTo, or Find.
// Subsequent calls return cached keys without re-reading.
// Returns an empty slice for null blocks; check IsNull for disambiguation.
func (sb *StreamBlock) ReadHeader() ([][]byte, error) {
	if sb.headerRead {
		return sb.keys, nil
	}

	// Read version(2B) + size(4B) + pairCount(2B) = 8 bytes.
	// We read the first 6 into headerBub
	sb.headerBuf = growBuf(sb.headerBuf, 8)
	if _, err := io.ReadFull(sb.r, sb.headerBuf[:6]); err != nil {
		return nil, err
	}

	// null block has size = 0 and contains no more data
	if binary.LittleEndian.Uint32(sb.headerBuf[2:6]) == 0 {
		sb.isNull = true
		sb.headerRead = true
		return sb.keys, nil
	}

	// Read pairCount, 2 bytes
	if _, err := io.ReadFull(sb.r, sb.headerBuf[6:8]); err != nil {
		return nil, err
	}

	n := int(binary.LittleEndian.Uint16(sb.headerBuf[6:8]))
	sb.pairsCount = n

	// Read value types (N bytes) + key offsets ((N+1)*2)
	metaSize := n + (n+1)*2
	sb.headerBuf = growBuf(sb.headerBuf, metaSize)
	if _, err := io.ReadFull(sb.r, sb.headerBuf[:metaSize]); err != nil {
		return nil, err
	}
	// Last 2 bytes of keyOffsets is the total key section size
	totalKeyBytes := int(binary.LittleEndian.Uint16(sb.headerBuf[metaSize-2:]))

	// Read the remainder
	reqSize := metaSize + totalKeyBytes + (n+1)*hValOffsetSize
	sb.headerBuf = growBuf(sb.headerBuf, reqSize)
	if _, err := io.ReadFull(sb.r, sb.headerBuf[metaSize:reqSize]); err != nil {
		return nil, err
	}

	sb.valueTypes = sb.headerBuf[:n]
	keyOffsetsRaw := sb.headerBuf[n:metaSize]
	keyBytes := sb.headerBuf[metaSize : metaSize+totalKeyBytes]
	valOffsetsRaw := sb.headerBuf[metaSize+totalKeyBytes : reqSize]

	// Parse keys
	for i := range n {
		startOff := int(binary.LittleEndian.Uint16(keyOffsetsRaw[i*2:]))
		endOff := int(binary.LittleEndian.Uint16(keyOffsetsRaw[(i+1)*2:]))
		sb.keys = append(sb.keys, keyBytes[startOff:endOff])
	}

	// Value sizes from offset pairs
	for i := range n {
		startOff := int(binary.LittleEndian.Uint32(valOffsetsRaw[i*hValOffsetSize:]))
		endOff := int(binary.LittleEndian.Uint32(valOffsetsRaw[(i+1)*hValOffsetSize:]))
		sb.valSizes = append(sb.valSizes, endOff-startOff)
	}

	sb.headerRead = true
	return sb.keys, nil
}

func (sb *StreamBlock) Keys() [][]byte {
	return sb.keys
}

func (sb *StreamBlock) NumPairs() int {
	return sb.pairsCount
}

func (sb *StreamBlock) TypeAt(i int) Type {
	return Type(sb.valueTypes[i])
}

// IsNull reports whether the current block is a null block (size == 0).
func (sb *StreamBlock) IsNull() bool {
	return sb.isNull
}

// NextMap returns a StreamBlock for reading the next *Map* value as a nested block.
// The returned StreamBlock reads directly from the parent's underlying reader.
// The parent's Next, and Skip automatically drain any unread bytes
// from the inner block before advancing.
func (sb *StreamBlock) NextMap() (*StreamBlock, error) {
	if err := sb.drainInner(); err != nil {
		return nil, err
	}
	if sb.nextValIndex >= sb.pairsCount {
		return nil, io.EOF
	}
	if t := Type(sb.valueTypes[sb.nextValIndex]); t != TypeMap {
		return nil, fmt.Errorf("raf: NextMap called on value of type %s", t)
	}

	size := sb.valSizes[sb.nextValIndex]
	sb.nextValIndex++

	if sb.innerReader == nil {
		sb.innerReader = &io.LimitedReader{R: sb.r, N: int64(size)}
	} else {
		sb.innerReader.R = sb.r
		sb.innerReader.N = int64(size)
	}
	if sb.innerStreamBlock == nil {
		sb.innerStreamBlock = NewStreamBlock(sb.innerReader)
	} else {
		sb.innerStreamBlock.Reset(sb.innerReader)
	}
	return sb.innerStreamBlock, nil
}

// NextArray returns a StreamArray for reading the next *Array* value
// as a sequence of elements. The returned StreamArray reads directly from
// the parent's underlying reader. The parent's Next and Skip automatically
// drain any unread bytes from the inner array before advancing.
// Returns io.EOF when values are exhausted.
func (sb *StreamBlock) NextArray() (*StreamArray, error) {
	if err := sb.drainInner(); err != nil {
		return nil, err
	}
	if sb.nextValIndex >= sb.pairsCount {
		return nil, io.EOF
	}
	if t := Type(sb.valueTypes[sb.nextValIndex]); t != TypeArray {
		return nil, fmt.Errorf("raf: NextArray called on value of type %s", t)
	}

	size := sb.valSizes[sb.nextValIndex]
	sb.nextValIndex++

	if sb.innerReader == nil {
		sb.innerReader = &io.LimitedReader{R: sb.r, N: int64(size)}
	} else {
		sb.innerReader.R = sb.r
		sb.innerReader.N = int64(size)
	}
	if sb.innerStreamArray == nil {
		sb.innerStreamArray = NewStreamArray(sb.innerReader)
	} else {
		sb.innerStreamArray.Reset(sb.innerReader)
	}
	return sb.innerStreamArray, nil
}

func (sb *StreamBlock) drainInner() error {
	if sb.innerReader == nil || sb.innerReader.N == 0 {
		return nil
	}
	_, err := io.Copy(io.Discard, sb.innerReader)
	return err
}

// Next reads the next value from the stream.
// The returned Value is only valid until the next call to Next.
// Returns io.EOF when values are exhausted.
func (sb *StreamBlock) Next() (Value, error) {
	if err := sb.drainInner(); err != nil {
		return Value{}, err
	}
	if sb.nextValIndex >= sb.pairsCount {
		return Value{}, io.EOF
	}

	size := sb.valSizes[sb.nextValIndex]
	valType := Type(sb.valueTypes[sb.nextValIndex])
	sb.nextValIndex++

	if size == 0 {
		return Value{Type: valType}, nil
	}

	sb.valueBuf = growBuf(sb.valueBuf, size)
	if _, err := io.ReadFull(sb.r, sb.valueBuf); err != nil {
		return Value{}, err
	}

	return Value{Type: valType, Data: sb.valueBuf}, nil
}

// Skip discards the next value without reading it into memory.
// Returns io.EOF when values are exhausted.
func (sb *StreamBlock) Skip() error {
	// TODO use Seek if underlying reader supports it for better performance
	if err := sb.drainInner(); err != nil {
		return err
	}
	if sb.nextValIndex >= sb.pairsCount {
		return io.EOF
	}

	size := sb.valSizes[sb.nextValIndex]
	sb.nextValIndex++

	if size == 0 {
		return nil
	}

	_, err := io.CopyN(io.Discard, sb.r, int64(size))
	return err
}

func growBuf(buf []byte, n int) []byte {
	if cap(buf) >= n {
		return buf[:n]
	}
	newBuf := make([]byte, n)
	copy(newBuf, buf)
	return newBuf
}

// StreamArray reads array elements sequentially from an io.Reader.
// After calling ReadHeader, elements are read in order using Next.
//
// The Value returned by Next is only valid until the next call to Next.
type StreamArray struct {
	r          io.Reader
	headerRead bool
	elemType   Type
	count      int
	elemSize   int   // for fixed-size element types
	elemSizes  []int // for dynamic-size element types

	nextIndex int

	headerBuf []byte
	valueBuf  []byte

	innerReader      *io.LimitedReader
	innerStreamBlock *StreamBlock
	innerStreamArray *StreamArray
}

// NewStreamArray creates a StreamArray that reads from r.
func NewStreamArray(r io.Reader) *StreamArray {
	return &StreamArray{r: r}
}

// Reset resets the StreamArray to read from a new reader.
func (sa *StreamArray) Reset(r io.Reader) {
	sa.r = r
	sa.headerRead = false
	sa.elemType = 0
	sa.count = 0
	sa.elemSize = 0
	sa.nextIndex = 0
	sa.elemSizes = sa.elemSizes[:0]
	sa.headerBuf = sa.headerBuf[:0]
	sa.valueBuf = sa.valueBuf[:0]
}

// ReadHeader reads the array header and must be called before Next.
// Subsequent calls return immediately without re-reading.
func (sa *StreamArray) ReadHeader() error {
	if sa.headerRead {
		return nil
	}

	// Read [u8 type][u16 count] = 3 bytes
	sa.headerBuf = growBuf(sa.headerBuf, 3)
	if _, err := io.ReadFull(sa.r, sa.headerBuf[:3]); err != nil {
		return err
	}

	sa.elemType = Type(sa.headerBuf[0])
	sa.count = int(binary.LittleEndian.Uint16(sa.headerBuf[1:3]))

	if sa.elemType.isDynamic() && sa.count > 0 {
		// Read (N+1)*2 offset bytes
		offsetsSize := (sa.count + 1) * 2
		sa.headerBuf = growBuf(sa.headerBuf, offsetsSize)
		if _, err := io.ReadFull(sa.r, sa.headerBuf[:offsetsSize]); err != nil {
			return err
		}

		for i := range sa.count {
			startOff := int(binary.LittleEndian.Uint16(sa.headerBuf[i*2:]))
			endOff := int(binary.LittleEndian.Uint16(sa.headerBuf[(i+1)*2:]))
			sa.elemSizes = append(sa.elemSizes, endOff-startOff)
		}
	} else {
		sa.elemSize = sa.elemType.Size()
	}

	sa.headerRead = true
	return nil
}

// ElemType returns the element type of the array.
// Must be called after ReadHeader.
func (sa *StreamArray) ElemType() Type {
	return sa.elemType
}

// Len returns the number of elements in the array.
// Must be called after ReadHeader.
func (sa *StreamArray) Len() int {
	return sa.count
}

func (sa *StreamArray) drainInner() error {
	if sa.innerReader == nil || sa.innerReader.N == 0 {
		return nil
	}
	_, err := io.Copy(io.Discard, sa.innerReader)
	return err
}

func (sa *StreamArray) nextElemSize() int {
	if sa.elemType.isDynamic() {
		return sa.elemSizes[sa.nextIndex]
	}
	return sa.elemSize
}

// Next reads the next element from the array.
// The returned Value is only valid until the next call to Next.
// Returns io.EOF when elements are exhausted.
func (sa *StreamArray) Next() (Value, error) {
	if err := sa.drainInner(); err != nil {
		return Value{}, err
	}
	if sa.nextIndex >= sa.count {
		return Value{}, io.EOF
	}

	size := sa.nextElemSize()
	sa.nextIndex++

	if size == 0 {
		return Value{Type: sa.elemType}, nil
	}

	sa.valueBuf = growBuf(sa.valueBuf, size)
	if _, err := io.ReadFull(sa.r, sa.valueBuf[:size]); err != nil {
		return Value{}, err
	}

	return Value{Type: sa.elemType, Data: sa.valueBuf[:size]}, nil
}

// NextMap returns a StreamBlock for reading the next array element as a nested block.
// The element type must be TypeMap.
// The returned StreamBlock is only valid until the next call on this StreamArray.
func (sa *StreamArray) NextMap() (*StreamBlock, error) {
	if err := sa.drainInner(); err != nil {
		return nil, err
	}
	if sa.nextIndex >= sa.count {
		return nil, io.EOF
	}
	if sa.elemType != TypeMap {
		return nil, fmt.Errorf("raf: StreamArray.NextMap called on array of type %s", sa.elemType)
	}

	size := sa.nextElemSize()
	sa.nextIndex++

	if sa.innerReader == nil {
		sa.innerReader = &io.LimitedReader{R: sa.r, N: int64(size)}
	} else {
		sa.innerReader.R = sa.r
		sa.innerReader.N = int64(size)
	}
	if sa.innerStreamBlock == nil {
		sa.innerStreamBlock = NewStreamBlock(sa.innerReader)
	} else {
		sa.innerStreamBlock.Reset(sa.innerReader)
	}
	return sa.innerStreamBlock, nil
}

// NextArray returns a StreamArray for reading the next array element as a nested array.
// The element type must be TypeArray.
// The returned StreamArray is only valid until the next call on this StreamArray.
func (sa *StreamArray) NextArray() (*StreamArray, error) {
	if err := sa.drainInner(); err != nil {
		return nil, err
	}
	if sa.nextIndex >= sa.count {
		return nil, io.EOF
	}
	if sa.elemType != TypeArray {
		return nil, fmt.Errorf("raf: StreamArray.NextArray called on array of type %s", sa.elemType)
	}

	size := sa.nextElemSize()
	sa.nextIndex++

	if sa.innerReader == nil {
		sa.innerReader = &io.LimitedReader{R: sa.r, N: int64(size)}
	} else {
		sa.innerReader.R = sa.r
		sa.innerReader.N = int64(size)
	}
	if sa.innerStreamArray == nil {
		sa.innerStreamArray = NewStreamArray(sa.innerReader)
	} else {
		sa.innerStreamArray.Reset(sa.innerReader)
	}
	return sa.innerStreamArray, nil
}
