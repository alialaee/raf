package raf

import (
	"encoding/binary"
	"errors"
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
	pairsCount int
	valueTypes []byte
	keys       [][]byte
	valSizes   []int //  byte size of each value

	nextValIndex int

	headerBuf []byte
	valueBuf  []byte
}

// NewStreamBlock creates a StreamBlock that reads from r.
func NewStreamBlock(r io.Reader) *StreamBlock {
	return &StreamBlock{r: r}
}

// Reset resets the StreamBlock to read from a new reader.
func (sb *StreamBlock) Reset(r io.Reader) {
	sb.r = r
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
func (sb *StreamBlock) ReadHeader() ([][]byte, error) {
	if sb.keys != nil {
		return sb.keys, nil
	}

	// Read version(2) + size(4) + pairCount(2) = 8 bytes
	var hdr [8]byte
	if _, err := io.ReadFull(sb.r, hdr[:]); err != nil {
		return nil, err
	}

	n := int(binary.LittleEndian.Uint16(hdr[6:8]))
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

// Next reads the next value from the stream.
// The returned Value.Data is only valid until the next call to Next.
// Returns io.EOF when values are exhausted.
func (sb *StreamBlock) Next() (Value, error) {
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
