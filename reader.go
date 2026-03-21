package raf

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Block is a zero-allocation reader for a RAF formatted byte slice.
// It is unsafe to use if Valid() returns false.
type Block struct {
	data       []byte
	pairsCount int
	keysBegin  int
	valsBegin  int
}

func NewBlock(data []byte) Block {
	if len(data) < 8 {
		return Block{data: data}
	}

	n := int(binary.LittleEndian.Uint16(data[6:8]))
	keyOffsetsBegin := 8 + n
	keysBegin := keyOffsetsBegin + (n+1)*2

	// Last key offset gives total key bytes length
	lastKeyOffPos := keyOffsetsBegin + n*2
	keysLength := int(binary.LittleEndian.Uint16(data[lastKeyOffPos : lastKeyOffPos+2]))

	valOffsetsBegin := keysBegin + keysLength
	valsBegin := valOffsetsBegin + (n+1)*hValOffsetSize

	return Block{
		data:       data,
		pairsCount: n,
		keysBegin:  keysBegin,
		valsBegin:  valsBegin,
	}
}

func (b *Block) Valid() bool {
	return true
}

// NumPairs returns the number of key-value pairs in the block.
func (b *Block) NumPairs() int {
	return b.pairsCount
}

// KeyAt returns the key bytes at the given index.
// It panics if i < 0 or i >= NumPairs().
func (b *Block) KeyAt(i int) []byte {
	return b.keyAt(i, b.keysBegin)
}

func (b *Block) keyAt(i int, keysBegin int) []byte {
	keyOffsetsBegin := 8 + b.pairsCount
	startOffsIdx := keyOffsetsBegin + i*2
	endOffsIdx := startOffsIdx + 2

	startOff := int(binary.LittleEndian.Uint16(b.data[startOffsIdx : startOffsIdx+2]))
	endOff := int(binary.LittleEndian.Uint16(b.data[endOffsIdx : endOffsIdx+2]))

	return b.data[keysBegin+startOff : keysBegin+endOff]
}

// ValueAt returns the value at the given index.
// It panics if i < 0 or i >= NumPairs().
func (b *Block) ValueAt(i int) Value {
	n := b.pairsCount
	valTypesBegin := 8
	valOffsetsBegin := b.valsBegin - (n+1)*hValOffsetSize

	return b.valueAt(i, valTypesBegin, valOffsetsBegin, b.valsBegin)
}

func (b *Block) valueAt(i int, valTypesBegin, valOffsetsBegin, valsBegin int) Value {
	valType := Type(b.data[valTypesBegin+i])

	startOffsIdx := valOffsetsBegin + i*hValOffsetSize
	endOffsIdx := startOffsIdx + hValOffsetSize

	startOff := int(binary.LittleEndian.Uint32(b.data[startOffsIdx : startOffsIdx+4]))
	endOff := int(binary.LittleEndian.Uint32(b.data[endOffsIdx : endOffsIdx+4]))

	return Value{
		Type: valType,
		Data: b.data[valsBegin+startOff : valsBegin+endOff],
	}
}

// Get performs a binary search to find the specified key.
// It returns the value and true if found.
func (b *Block) Get(key []byte) (Value, bool) {
	n := b.pairsCount
	if n == 0 {
		return Value{}, false
	}

	valTypesBegin := 8
	valOffsetsBegin := b.valsBegin - (n+1)*hValOffsetSize
	keysBegin := b.keysBegin
	valsBegin := b.valsBegin

	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if bytes.Compare(b.keyAt(h, keysBegin), key) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}

	if i < n && bytes.Equal(b.keyAt(i, keysBegin), key) {
		return b.valueAt(i, valTypesBegin, valOffsetsBegin, valsBegin), true
	}

	return Value{}, false
}

// Array is a zero-allocation reader for an array value.
// It reads from the raw value bytes returned by Block.ValueAt or Block.Get
// when the value type is TypeArray.
type Array []byte

// ElemType returns the element type of the array.
func (a Array) ElemType() Type {
	return Type(a[0])
}

func (a Array) Len() int {
	return int(binary.LittleEndian.Uint16(a[1:3]))
}

// At returns the raw bytes for the element at index i.
// For fixed-size element types, it uses arithmetic indexing.
// For dynamically-sized types (e.g. string), it reads from the offset table.
// It panics if i < 0 or i >= Len().
func (a Array) At(i int) []byte {
	elemType := a.ElemType()

	if elemType.isDynamic() {
		// Layout: [u8 type][u16 count][u16*(N+1) offsets][...data]
		n := a.Len()
		dataStart := 3 + (n+1)*2

		startOffsIdx := 3 + i*2
		endOffsIdx := startOffsIdx + 2

		startOff := int(binary.LittleEndian.Uint16(a[startOffsIdx : startOffsIdx+2]))
		endOff := int(binary.LittleEndian.Uint16(a[endOffsIdx : endOffsIdx+2]))

		return a[dataStart+startOff : dataStart+endOff]
	}

	// Fixed-size: Layout: [u8 type][u16 count][...data]
	elemSize := elemType.Size()
	start := 3 + i*elemSize
	return a[start : start+elemSize]
}

func (a Array) AtString(i int, buf []byte) []byte {
	if a.ElemType() != TypeString {
		return nil
	}
	return append(buf[0:0], a.At(i)...)
}

func (a Array) AtInt64(i int) int64 {
	if a.ElemType() != TypeInt64 {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(a.At(i)))
}

func (a Array) AtFloat64(i int) float64 {
	if a.ElemType() != TypeFloat64 {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(a.At(i)))
}

func (a Array) AtBool(i int) bool {
	if a.ElemType() != TypeBool {
		return false
	}
	return a.At(i)[0] != 0
}

func (a Array) AtMap(i int) Block {
	if a.ElemType() != TypeMap {
		return Block{}
	}
	return NewBlock(a.At(i))
}

// Value represents a typed value read from a RAF block.
// It is valid only as long as the underlying block byte slice is valid.
type Value struct {
	Type Type
	Data []byte
}

// Bytes uses buf to return the value as a byte slice without additional allocations.
func (v Value) Bytes(buf []byte) []byte {
	if v.Type != TypeString {
		return nil
	}
	return append(buf[0:0], v.Data...)
}

func (v Value) String() string {
	if v.Type != TypeString {
		return ""
	}
	return string(v.Data)
}

func (v Value) Int64() int64 {
	if v.Type != TypeInt64 || len(v.Data) != 8 {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(v.Data))
}

func (v Value) Float64() float64 {
	if v.Type != TypeFloat64 || len(v.Data) != 8 {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(v.Data))
}

func (v Value) Bool() bool {
	if v.Type != TypeBool || len(v.Data) == 0 {
		return false
	}
	return v.Data[0] != 0
}

func (v Value) Array() Array {
	if v.Type != TypeArray {
		return nil
	}
	return Array(v.Data)
}

func (v Value) Map() Block {
	if v.Type != TypeMap {
		return Block{}
	}
	return NewBlock(v.Data)
}
