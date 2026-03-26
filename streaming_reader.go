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

	headerBuf []byte
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
	sb.keys = sb.keys[:0]
	sb.valSizes = sb.valSizes[:0]
	sb.headerBuf = sb.headerBuf[:0]
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

func (sb *StreamBlock) Next() (Value, error) {
	return Value{}, nil
}

func (sb *StreamBlock) Skip() error {
	return nil
}

func growBuf(buf []byte, n int) []byte {
	if cap(buf) >= n {
		return buf[:n]
	}
	newBuf := make([]byte, n)
	copy(newBuf, buf)
	return newBuf
}
