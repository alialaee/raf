package raf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type Type uint8

const (
	TypeString  Type = 0x01
	TypeInt64   Type = 0x02
	TypeFloat64 Type = 0x03
	TypeBool    Type = 0x04
	TypeArray   Type = 0x05
	TypeMap     Type = 0x06
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
	case TypeArray:
		return "array"
	case TypeMap:
		return "map"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}
func (t Type) Size() int {
	switch t {
	case TypeString, TypeArray, TypeMap:
		return -1
	case TypeInt64, TypeFloat64:
		return 8
	case TypeBool:
		return 1
	default:
		panic(fmt.Sprintf("unknown type: %d", t))
	}
}

func (t Type) isDynamic() bool {
	return t.Size() < 0
}

var (
	ErrValueCountMismatch = errors.New("number of values added does not match number of keys")
	ErrArrayCountMismatch = errors.New("number of array elements added does not match expected count")
)

const (
	Version        = 0xff01
	hVersionSize   = 2
	hSizeSize      = 4 // u32
	hValOffsetSize = 4 // u32
)

type Builder struct {
	buf []byte

	valOffsetsStart int
	lastValOffset   int
	valueIndex      int
	keyCount        int

	// Cache for builders
	cachedArrayBuilder *ArrayBuilder
	cachedBuilder      *Builder
}

type KeyType struct {
	Name string
	Type Type
}

func NewBuilder(buf []byte) *Builder {
	b := &Builder{
		buf: buf,
	}
	b.Reset()

	return b
}

// Reset clears the builder state for zero-allocation reuse.
func (b *Builder) Reset() {
	b.buf = b.buf[:0]

	b.lastValOffset = 0
	b.valueIndex = 0
	b.keyCount = 0

	// Write version and reserve space for size
	if cap(b.buf) < hVersionSize+hSizeSize {
		b.buf = make([]byte, hVersionSize+hSizeSize)
	} else {
		b.buf = b.buf[:hVersionSize+hSizeSize]
	}
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

func (b *Builder) AddNull() {
	// Add an offset of 0 for null values. The reader will interpret this as null.
	binary.LittleEndian.PutUint32(b.buf[b.valOffsetsStart+b.valueIndex*hValOffsetSize:], uint32(b.lastValOffset))
	b.valueIndex++

}

func (b *Builder) AddArrayFn(elemType Type, count int, fn func(ab *ArrayBuilder) error) error {
	if b.cachedArrayBuilder == nil {
		b.cachedArrayBuilder = NewArrayBuilder(make([]byte, 0, 256), elemType, count)
	} else {
		b.cachedArrayBuilder.Reset(elemType, count)
	}
	if err := fn(b.cachedArrayBuilder); err != nil {
		return err
	}
	arr, err := b.cachedArrayBuilder.Build()
	if err != nil {
		return err
	}
	b.AddRaw(arr)
	return nil
}

func (b *Builder) AddBuilderFn(fn func(mb *Builder) error) error {
	if b.cachedBuilder == nil {
		b.cachedBuilder = NewBuilder(make([]byte, 0, 256))
	} else {
		b.cachedBuilder.Reset()
	}
	if err := fn(b.cachedBuilder); err != nil {
		return err
	}
	m, err := b.cachedBuilder.Build()
	if err != nil {
		return err
	}
	b.AddRaw(m)
	return nil
}

func (b *Builder) Build() ([]byte, error) {
	if b.valueIndex != b.keyCount {
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrValueCountMismatch, b.keyCount, b.valueIndex)
	}

	// Write the final value offset
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

// AddStringArray it's a helper function to add a string array to the builder.
// It's not the most performant way to add a string array to the builder.
func (b *Builder) AddStringArray(values ...string) error {
	return b.AddArrayFn(TypeString, len(values), func(ab *ArrayBuilder) error {
		for _, v := range values {
			ab.AddString(v)
		}
		return nil
	})
}

func (b *Builder) AddBoolArray(values ...bool) error {
	return b.AddArrayFn(TypeBool, len(values), func(ab *ArrayBuilder) error {
		for _, v := range values {
			ab.AddBool(v)
		}
		return nil
	})
}

func (b *Builder) AddInt64Array(values ...int64) error {
	return b.AddArrayFn(TypeInt64, len(values), func(ab *ArrayBuilder) error {
		for _, v := range values {
			ab.AddInt64(v)
		}
		return nil
	})
}

func (b *Builder) AddFloat64Array(values ...float64) error {
	return b.AddArrayFn(TypeFloat64, len(values), func(ab *ArrayBuilder) error {
		for _, v := range values {
			ab.AddFloat64(v)
		}
		return nil
	})
}

const (
	aTypeSize   = 1 // u8
	aCountSize  = 2 // u16
	aOffsetSize = 2 // u16
)

type ArrayBuilder struct {
	buf          []byte
	elemType     Type
	count        int
	valueIndex   int
	isDynamic    bool
	offsetsStart int
	lastOffset   int

	builderCache      *Builder
	arrayBuilderCache *ArrayBuilder
}

func NewArrayBuilder(buf []byte, elemType Type, count int) *ArrayBuilder {
	ab := &ArrayBuilder{buf: buf}
	ab.Reset(elemType, count)
	return ab
}

func (a *ArrayBuilder) Reset(elemType Type, count int) {
	a.buf = a.buf[:0]
	a.elemType = elemType
	a.count = count
	a.valueIndex = 0
	a.lastOffset = 0
	a.isDynamic = elemType == TypeString || elemType == TypeArray || elemType == TypeMap

	headerSize := aTypeSize + aCountSize
	if a.isDynamic {
		headerSize += (count + 1) * aOffsetSize
	}

	if cap(a.buf) < headerSize {
		a.buf = make([]byte, headerSize)
	} else {
		a.buf = a.buf[:headerSize]
	}

	a.buf[0] = byte(elemType)
	binary.LittleEndian.PutUint16(a.buf[aTypeSize:], uint16(count))

	if a.isDynamic {
		a.offsetsStart = aTypeSize + aCountSize
	}
}

func (a *ArrayBuilder) AddString(value string) {
	binary.LittleEndian.PutUint16(a.buf[a.offsetsStart+a.valueIndex*aOffsetSize:], uint16(a.lastOffset))
	a.valueIndex++
	a.buf = append(a.buf, value...)
	a.lastOffset += len(value)
}

func (a *ArrayBuilder) AddInt64(value int64) {
	a.valueIndex++
	pos := len(a.buf)
	a.buf = append(a.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(a.buf[pos:], uint64(value))
}

func (a *ArrayBuilder) AddFloat64(value float64) {
	a.valueIndex++
	pos := len(a.buf)
	a.buf = append(a.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(a.buf[pos:], math.Float64bits(value))
}

func (a *ArrayBuilder) AddBool(value bool) {
	a.valueIndex++
	if value {
		a.buf = append(a.buf, 1)
	} else {
		a.buf = append(a.buf, 0)
	}
}

func (a *ArrayBuilder) AddNull() {
	// Add an offset of 0 for null values. The reader will interpret this as null.
	binary.LittleEndian.PutUint16(a.buf[a.offsetsStart+a.valueIndex*aOffsetSize:], uint16(a.lastOffset))
	a.valueIndex++
}

func (a *ArrayBuilder) AddRaw(value []byte) {
	if a.isDynamic {
		binary.LittleEndian.PutUint16(a.buf[a.offsetsStart+a.valueIndex*aOffsetSize:], uint16(a.lastOffset))
		a.lastOffset += len(value)
	}
	a.valueIndex++
	a.buf = append(a.buf, value...)
}

func (a *ArrayBuilder) AddBuilderFn(fn func(mb *Builder) error) error {
	if a.builderCache == nil {
		a.builderCache = NewBuilder(make([]byte, 0, 256))
	} else {
		a.builderCache.Reset()
	}
	if err := fn(a.builderCache); err != nil {
		return err
	}
	m, err := a.builderCache.Build()
	if err != nil {
		return err
	}
	a.AddRaw(m)
	return nil
}

func (a *ArrayBuilder) AddArrayFn(elemType Type, count int, fn func(ab *ArrayBuilder) error) error {
	if a.arrayBuilderCache == nil {
		a.arrayBuilderCache = NewArrayBuilder(make([]byte, 0, 256), elemType, count)
	}
	a.arrayBuilderCache.Reset(elemType, count)
	if err := fn(a.arrayBuilderCache); err != nil {
		return err
	}
	arr, err := a.arrayBuilderCache.Build()
	if err != nil {
		return err
	}
	a.AddRaw(arr)
	return nil
}

func (a *ArrayBuilder) Build() ([]byte, error) {
	if a.valueIndex != a.count {
		return nil, fmt.Errorf("%w: expected %d elements, got %d", ErrArrayCountMismatch, a.count, a.valueIndex)
	}
	if a.isDynamic {
		binary.LittleEndian.PutUint16(a.buf[a.offsetsStart+a.count*aOffsetSize:], uint16(a.lastOffset))
	}
	return a.buf, nil
}
