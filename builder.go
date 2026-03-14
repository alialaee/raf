package raf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

type Type uint8

const (
	Version uint8 = 0x01

	TypeString  Type = 0x01
	TypeInt64   Type = 0x02
	TypeFloat64 Type = 0x03
	TypeBool    Type = 0x04
	TypeNull    Type = 0xff
	TypeArray   Type = 0x05
	TypeMap     Type = 0x06

	maxKeySize = 4 * 1024
)

func (t Type) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt64:
		return "int64"
	case TypeFloat64:
		return "float64"
	case TypeBool:
		return "bool"
	case TypeNull:
		return "null"
	case TypeArray:
		return "array"
	case TypeMap:
		return "map"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

var (
	ErrBlockTooLarge = errors.New("raf: block too large (max 64KB)")
	ErrTooManyPairs  = errors.New("raf: too many pairs (max 255)")
	ErrInvalidKey    = errors.New("raf: key is not valid")
)

func (t Type) Size() int {
	switch t {
	case TypeString, TypeArray, TypeMap:
		return -1
	case TypeInt64, TypeFloat64:
		return 8
	case TypeBool:
		return 1
	case TypeNull:
		return 0
	default:
		panic(fmt.Sprintf("unknown type: %d", t))
	}
}

func (t Type) isDynamic() bool {
	return t.Size() < 0
}

// Builder allows for zero-allocation encoding of raf blocks.
type Builder struct {
	keys       []byte
	vals       []byte
	keyOffsets []uint16 // Starts with 0
	valOffsets []uint16 // Starts with 0
	types      []byte

	lastKey       string   // Tracked to ensure keys are added in sorted order
	arrayBuf      []byte   // Scratch buffer reused for array value serialization
	inner         *Builder // Reusable inner builder for nested map and struct encoding
	innerBuildBuf []byte   // Scratch buffer for inner.Build output and it's preserved across Reset
}

func NewBuilder() *Builder {
	b := &Builder{}
	b.Reset()
	return b
}

// Reset clears the builder state for zero-allocation reuse.
func (b *Builder) Reset() {
	b.keys = b.keys[:0]
	b.vals = b.vals[:0]

	// Reset offsets to contain only the 0th offset
	b.keyOffsets = b.keyOffsets[:0]
	b.keyOffsets = append(b.keyOffsets, 0)
	b.valOffsets = b.valOffsets[:0]
	b.valOffsets = append(b.valOffsets, 0)

	b.types = b.types[:0]
	b.lastKey = ""
	b.arrayBuf = b.arrayBuf[:0]
}

// checkKey verifies that the key is strictly greater than the last added key.
func (b *Builder) checkKey(key string) error {
	keySize := len(key)
	if keySize == 0 {
		return fmt.Errorf("%w: keys should be larger than zero", ErrInvalidKey)
	}
	if keySize > maxKeySize {
		return fmt.Errorf("%w: keys should be smaller than 4KB", ErrInvalidKey)
	}

	if len(b.keys) > 0 {
		cmp := strings.Compare(key, b.lastKey)
		if cmp == 0 {
			return fmt.Errorf("%w: duplicate key", ErrInvalidKey)
		}
		if cmp < 0 {
			return fmt.Errorf("%w: keys not added in lexicographical order", ErrInvalidKey)
		}
	}

	if len(b.types) >= 255 {
		return ErrTooManyPairs
	}

	return nil
}

func (b *Builder) appendKey(key string) {
	b.keys = append(b.keys, key...)
	b.keyOffsets = append(b.keyOffsets, uint16(len(b.keys)))

	b.lastKey = key
}

func (b *Builder) appendValue(valType Type, valBytes []byte) {
	b.vals = append(b.vals, valBytes...)
	b.valOffsets = append(b.valOffsets, uint16(len(b.vals)))
	b.types = append(b.types, uint8(valType))
}

func (b *Builder) AddString(key string, val []byte) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)
	b.appendValue(TypeString, val)
	return nil
}

// AddStringString is a helper for adding string values without
// unnecessary byte slice conversions.
func (b *Builder) AddStringString(key string, val string) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	b.vals = append(b.vals, val...)
	b.valOffsets = append(b.valOffsets, uint16(len(b.vals)))
	b.types = append(b.types, byte(TypeString))

	return nil
}

func (b *Builder) AddInt64(key string, val int64) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(val))
	b.appendValue(TypeInt64, buf[:])

	return nil
}

func (b *Builder) AddFloat64(key string, val float64) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
	b.appendValue(TypeFloat64, buf[:])

	return nil
}

func (b *Builder) AddBool(key string, val bool) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	if val {
		b.vals = append(b.vals, 0x01)
	} else {
		b.vals = append(b.vals, 0x00)
	}
	b.valOffsets = append(b.valOffsets, uint16(len(b.vals)))
	b.types = append(b.types, uint8(TypeBool))

	return nil
}

func (b *Builder) AddNull(key string) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)
	b.appendValue(TypeNull, nil)

	return nil
}

// AddMap adds a map value. val must be a pre-built block (from Builder.Build).
func (b *Builder) AddMap(key string, val []byte) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)
	b.appendValue(TypeMap, val)
	return nil
}

// AddMapFn writes a nested map/struct value by calling fn with a reusable inner Builder.
// The inner Builder is reset before fn is called and must not be retained after fn returns.
// After warmup, this method incurs zero allocations.
func (b *Builder) AddMapFn(key string, fn func(inner *Builder) error) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)
	if b.inner == nil {
		b.inner = &Builder{}
	}
	b.inner.Reset()
	if err := fn(b.inner); err != nil {
		return err
	}
	var err error
	b.innerBuildBuf, err = b.inner.Build(b.innerBuildBuf)
	if err != nil {
		return err
	}
	b.appendValue(TypeMap, b.innerBuildBuf)
	return nil
}

// addMapArrayFromFn writes a TypeArray(TypeMap) value where each element is built by fn.
// fn is called once per element; its inner Builder is reset between calls.
// A single inner Builder is reused for all elements, avoiding per-element allocations.
func (b *Builder) addMapArrayFromFn(key string, count int, fn func(i int, inner *Builder) error) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)
	if b.inner == nil {
		b.inner = &Builder{}
	}

	b.appendArrayHeader(TypeMap, count)

	offsetStart := len(b.arrayBuf)
	for range count + 1 {
		b.arrayBuf = append(b.arrayBuf, 0, 0)
	}

	var off uint16
	var buf [2]byte
	for i := range count {
		b.inner.Reset()
		if err := fn(i, b.inner); err != nil {
			return err
		}
		var err error
		b.innerBuildBuf, err = b.inner.Build(b.innerBuildBuf)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(buf[:], off)
		copy(b.arrayBuf[offsetStart+i*2:], buf[:])
		b.arrayBuf = append(b.arrayBuf, b.innerBuildBuf...)
		off += uint16(len(b.innerBuildBuf))
	}
	binary.LittleEndian.PutUint16(buf[:], off)
	copy(b.arrayBuf[offsetStart+count*2:], buf[:])

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}

// appendArrayHeader writes the element type (u8) + count (u16) to arrayBuf.
func (b *Builder) appendArrayHeader(elemType Type, count int) {
	b.arrayBuf = b.arrayBuf[:0]
	b.arrayBuf = append(b.arrayBuf, byte(elemType))
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(count))
	b.arrayBuf = append(b.arrayBuf, buf[:]...)
}

func (b *Builder) AddStringArray(key string, vals [][]byte) error {
	return addStringArray(b, key, vals)
}

// AddStringStringArray is a helper for adding arrays of strings without
// unnecessary byte slice conversions.
func (b *Builder) AddStringStringArray(key string, vals []string) error {
	return addStringArray(b, key, vals)
}

func (b *Builder) AddInt64Array(key string, vals []int64) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	b.appendArrayHeader(TypeInt64, len(vals))

	var buf [8]byte
	for _, v := range vals {
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		b.arrayBuf = append(b.arrayBuf, buf[:]...)
	}

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}

func (b *Builder) AddFloat64Array(key string, vals []float64) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	b.appendArrayHeader(TypeFloat64, len(vals))

	var buf [8]byte
	for _, v := range vals {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
		b.arrayBuf = append(b.arrayBuf, buf[:]...)
	}

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}

func (b *Builder) AddBoolArray(key string, vals []bool) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	b.appendArrayHeader(TypeBool, len(vals))

	for _, v := range vals {
		if v {
			b.arrayBuf = append(b.arrayBuf, 0x01)
		} else {
			b.arrayBuf = append(b.arrayBuf, 0x00)
		}
	}

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}

// AddMapArray adds an array of pre-built map blocks.
// Each element of vals must be a valid RAF block (from Builder.Build).
func (b *Builder) AddMapArray(key string, vals [][]byte) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	b.appendArrayHeader(TypeMap, len(vals))

	// Offset table: (N+1) u16 values
	offsetStart := len(b.arrayBuf)
	for range len(vals) + 1 {
		b.arrayBuf = append(b.arrayBuf, 0, 0)
	}

	var off uint16
	var buf [2]byte
	for i, v := range vals {
		binary.LittleEndian.PutUint16(buf[:], off)
		copy(b.arrayBuf[offsetStart+i*2:], buf[:])
		b.arrayBuf = append(b.arrayBuf, v...)
		off += uint16(len(v))
	}
	// Final sentinel offset
	binary.LittleEndian.PutUint16(buf[:], off)
	copy(b.arrayBuf[offsetStart+len(vals)*2:], buf[:])

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}

// EstimateSize calculates the exact number of bytes the block will consume.
func (b *Builder) EstimateSize() int {
	n := len(b.types)
	return 1 + // Version (u8)
		2 + // Total data size (u16)
		1 + // Number of pairs (u8)
		(n+1)*2 + // key offsets (u16)
		n*1 + // value types (u8)
		(n+1)*2 + // value offsets (u16)
		len(b.keys) + // key bytes
		len(b.vals) // value bytes
}

// Build serializes the block to the given destination byte slice.
// If cap(dst) is large enough, this performs zero allocations.
// dst can be nil. It returns the filled byte slice and any formatting errors.
func (b *Builder) Build(dst []byte) ([]byte, error) {
	size := b.EstimateSize()
	if size > math.MaxUint16 {
		return nil, ErrBlockTooLarge
	}

	if cap(dst) < size {
		dst = make([]byte, size)
	} else {
		dst = dst[:size]
	}

	// Layout:
	//	[u8]  Version (e.g., 0x01)
	//	[u16] Total data size
	//	[u8]  Number of pairs (N)
	//	[u16 * (N+1)] Array of key offsets (relative to start of key bytes)
	//	[u8  * N]     Array of value types
	//	[u16 * (N+1)] Array of value offsets (relative to start of value bytes)
	//	[...u8]       Array of key bytes
	//	[...u8]       Array of value bytes

	cursor := 0

	// 1. Version
	dst[cursor] = Version
	cursor += 1

	// 2. Total data size
	binary.LittleEndian.PutUint16(dst[cursor:], uint16(size))
	cursor += 2

	// 3. Number of pairs (N)
	dst[cursor] = uint8(len(b.types))
	cursor += 1

	// 4. Array of key offsets
	for _, offset := range b.keyOffsets {
		binary.LittleEndian.PutUint16(dst[cursor:], offset)
		cursor += 2
	}

	// 5. Array of value types
	copy(dst[cursor:], b.types)
	cursor += len(b.types)

	// 6. Array of value offsets
	for _, offset := range b.valOffsets {
		binary.LittleEndian.PutUint16(dst[cursor:], offset)
		cursor += 2
	}

	// 7. Array of key bytes
	copy(dst[cursor:], b.keys)
	cursor += len(b.keys)

	// 8. Array of value bytes
	copy(dst[cursor:], b.vals)

	return dst, nil
}

func addStringArray[T interface{ string | []byte }](b *Builder, key string, vals []T) error {
	if err := b.checkKey(key); err != nil {
		return err
	}
	b.appendKey(key)

	// Header: type + count
	b.appendArrayHeader(TypeString, len(vals))

	// Offset table: (N+1) u16 values
	// First pass: compute offsets and append placeholders
	offsetStart := len(b.arrayBuf)
	for range len(vals) + 1 {
		b.arrayBuf = append(b.arrayBuf, 0, 0)
	}

	// Second pass: write values and fill offsets
	var off uint16
	var buf [2]byte
	for i, v := range vals {
		binary.LittleEndian.PutUint16(buf[:], off)
		copy(b.arrayBuf[offsetStart+i*2:], buf[:])
		b.arrayBuf = append(b.arrayBuf, v...)
		off += uint16(len(v))
	}
	// Final sentinel offset
	binary.LittleEndian.PutUint16(buf[:], off)
	copy(b.arrayBuf[offsetStart+len(vals)*2:], buf[:])

	b.appendValue(TypeArray, b.arrayBuf)
	return nil
}
