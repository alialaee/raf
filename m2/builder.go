package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Package raf provides a simple, read-optimized binary format for key-value pairs.
// Keys are sorted lexicographically as raw bytes to allow fast retrieval.
// It's designed for a few kilobytes of data, with a focus on fast read access.
// All integer fields are stored in big-endian format.
//
// Layout:
//
//	[u8]  Version (e.g., 0x01)
//	[u32] Total data size
//	[u16]  Number of pairs (N)
//	[u8  * N]     Array of value types
//	[u16 * (N+1)] Array of key offsets (relative to start of key bytes)
//	[...u8]       Array of key bytes
//	[u32 * (N+1)] Array of value offsets (relative to start of value bytes)
//	[...u8]       Array of value bytes
//
// Value Types (1 byte):
//
//	0x01: string      0x04: bool
//	0x02: int64       0x05: array
//	0x03: float64     0x06: map (value is the same as the Layout, it's recursive)
//
// Arrays:
//
//	[u8] Type of array elements (same as value types above)
//	[u16] Entries in the array
//	[u16 * (N+1)] Offsets for the values only if the type is dynamically sized (e.g., string).
//
// Notes:
// - Keys: Must be unique, raw byte arrays, and ordered by byte value (not locale-aware).
// - Strings: Raw byte arrays. Decoding (e.g., UTF-8) is left to the client. Zero-length strings are permitted.
// - Booleans: 1 byte long. 0x00 is false, any other value is true.

type Type uint8

const (
	TypeString  Type = 0x01
	TypeInt64   Type = 0x02
	TypeFloat64 Type = 0x03
	TypeBool    Type = 0x04
	TypeArray   Type = 0x05
	TypeMap     Type = 0x06
)

var (
	ErrValueCountMismatch = errors.New("number of values added does not match number of keys")
)

const (
	Version        byte = 0x01
	hVersionSize        = 1
	hSizeSize           = 4 // u32
	hCountSize          = 2 // u16
	hKeyOffsetSize      = 2 // u16
	hValTypeSize        = 1 // u8
	hValOffsetSize      = 4 // u32
	hSize               = hVersionSize + hSizeSize + hCountSize
)

type Builder struct {
	buf []byte

	valOffsetsStart int
	lastValOffset   int
	valueIndex      int
	keyCount        int
}

func NewBuilder(buf []byte) *Builder {
	b := &Builder{
		buf: buf,
	}
	b.Reset()

	return b
}

func (b *Builder) Reset() {
	b.buf = b.buf[:0]

	b.lastValOffset = 0
	b.valueIndex = 0
	b.keyCount = 0

	b.buf = append(b.buf, Version)
	// Reserve space for size
	b.buf = append(b.buf, 0, 0, 0, 0) // size (u32)
}

type KeyType struct {
	Name string
	Type Type
}

func (b *Builder) AddKeys(keys ...KeyType) {
	count := len(keys)
	b.keyCount = count

	var keyBytesLen int
	for i := range count {
		keyBytesLen += len(keys[i].Name)
	}

	totalAdded := 2 + count + (count+1)*2 + keyBytesLen + (count+1)*hValOffsetSize
	start := len(b.buf)

	// Just ensure we have enough capacity to write without appends
	if cap(b.buf)-start < totalAdded {
		newBuf := make([]byte, start, start*2+totalAdded)
		copy(newBuf, b.buf)
		b.buf = newBuf
	}
	b.buf = b.buf[:start+totalAdded]

	pos := start

	// Add key count
	binary.LittleEndian.PutUint16(b.buf[pos:], uint16(count))
	pos += 2

	// Add value types
	for i := range count {
		b.buf[pos] = byte(keys[i].Type)
		pos++
	}

	// Add key offsets
	offset := 0
	for i := range count {
		binary.LittleEndian.PutUint16(b.buf[pos:], uint16(offset))
		keySize := len(keys[i].Name)

		offset += keySize
		pos += 2
	}
	binary.LittleEndian.PutUint16(b.buf[pos:], uint16(offset)) // For the end of the last key
	pos += 2

	// Add key bytes
	for i := range count {
		n := copy(b.buf[pos:], keys[i].Name)
		pos += n
	}

	// Value offsets space is already reserved in totalAdded
	b.valOffsetsStart = pos
}

func (b *Builder) AddString(value string) {
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++
	b.buf = append(b.buf, value...)
	b.lastValOffset += len(value)
}

func (b *Builder) AddInt64(value int64) {
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++
	pos := len(b.buf)
	b.buf = append(b.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(b.buf[pos:], uint64(value))
	b.lastValOffset += 8
}

func (b *Builder) AddFloat64(value float64) {
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++
	pos := len(b.buf)
	b.buf = append(b.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(b.buf[pos:], math.Float64bits(value))
	b.lastValOffset += 8
}

func (b *Builder) AddBool(value bool) {
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++
	if value {
		b.buf = append(b.buf, 1)
	} else {
		b.buf = append(b.buf, 0)
	}
	b.lastValOffset++
}

func (b *Builder) Build() ([]byte, error) {
	if b.valueIndex != b.keyCount {
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrValueCountMismatch, b.keyCount, b.valueIndex)
	}

	// Write the final value offset (end sentinel)
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.keyCount*hValOffsetSize:], uint32(b.lastValOffset))

	// Update total size
	totalSize := len(b.buf) - hVersionSize - hSizeSize
	binary.LittleEndian.PutUint32(b.buf[hVersionSize:hVersionSize+hSizeSize], uint32(totalSize))

	return b.buf, nil
}

// AddRaw allows adding a pre-serialized value directly to the builder.
// This is for for adding arrays or maps where the value is already in the correct format.
func (b *Builder) AddRaw(value []byte) {
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++
	b.buf = append(b.buf, value...)
	b.lastValOffset += len(value)
}

type ArrayBulder struct {
	buf []byte
}
