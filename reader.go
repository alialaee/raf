package raf

import (
	"bytes"
	"encoding/binary"
)

// Block is a zero-allocation reader for a RAF formatted byte slice.
// It is unsafe to use if Valid() returns false.
type Block []byte

// Valid does *NOT* thoroughly validate as it written to be
// fast.
func (b Block) Valid() bool {
	if len(b) < 4 {
		return false
	}
	if b[0] != Version {
		return false
	}
	size := binary.BigEndian.Uint16(b[1:3])
	return int(size) == len(b)
}

func (b Block) NumPairs() int {
	if len(b) < 4 {
		return 0
	}
	return int(b[3])
}

// KeyAt returns the key bytes at the given index.
// It panics if i < 0 or i >= NumPairs().
func (b Block) KeyAt(i int) []byte {
	n := b.NumPairs()
	keysBegin := 4 + (n+1)*2 + n + (n+1)*2
	return b.keyAt(i, keysBegin)
}

func (b Block) keyAt(i int, keysBegin int) []byte {
	startOffsIdx := 4 + (i * 2)
	endOffsIdx := startOffsIdx + 2

	startOff := int(binary.BigEndian.Uint16(b[startOffsIdx : startOffsIdx+2]))
	endOff := int(binary.BigEndian.Uint16(b[endOffsIdx : endOffsIdx+2]))

	return b[keysBegin+startOff : keysBegin+endOff]
}

// ValueAt returns the value type and raw value bytes at the given index.
// It panics if i < 0 or i >= NumPairs().
func (b Block) ValueAt(i int) (valType Type, val []byte) {
	n := b.NumPairs()
	valTypesBegin := 4 + (n+1)*2
	valOffsetsBegin := valTypesBegin + n
	keysBegin := valOffsetsBegin + (n+1)*2

	// Total key length from last key offset entry
	keysLength := int(binary.BigEndian.Uint16(b[4+n*2 : 4+n*2+2]))
	valsBegin := keysBegin + keysLength

	return b.valueAt(i, valTypesBegin, valOffsetsBegin, valsBegin)
}

func (b Block) valueAt(i int, valTypesBegin, valOffsetsBegin, valsBegin int) (Type, []byte) {
	valType := Type(b[valTypesBegin+i])

	startOffsIdx := valOffsetsBegin + (i * 2)
	endOffsIdx := startOffsIdx + 2

	startOff := int(binary.BigEndian.Uint16(b[startOffsIdx : startOffsIdx+2]))
	endOff := int(binary.BigEndian.Uint16(b[endOffsIdx : endOffsIdx+2]))

	return valType, b[valsBegin+startOff : valsBegin+endOff]
}

// Get performs a binary search to find the specified key.
// It returns the value type, the zero-allocated raw value bytes, and true if found.
func (b Block) Get(key []byte) (valType Type, val []byte, ok bool) {
	n := int(b[3])
	if n == 0 {
		return 0, nil, false
	}

	// Precompute all layout positions once.
	valTypesBegin := 4 + (n+1)*2
	valOffsetsBegin := valTypesBegin + n
	keysBegin := valOffsetsBegin + (n+1)*2
	keysLength := int(binary.BigEndian.Uint16(b[4+n*2 : 4+n*2+2]))
	valsBegin := keysBegin + keysLength

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
		vt, vb := b.valueAt(i, valTypesBegin, valOffsetsBegin, valsBegin)
		return vt, vb, true
	}

	return 0, nil, false
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
	return int(binary.BigEndian.Uint16(a[1:3]))
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

		startOff := int(binary.BigEndian.Uint16(a[startOffsIdx : startOffsIdx+2]))
		endOff := int(binary.BigEndian.Uint16(a[endOffsIdx : endOffsIdx+2]))

		return a[dataStart+startOff : dataStart+endOff]
	}

	// Fixed-size: Layout: [u8 type][u16 count][...data]
	elemSize := elemType.Size()
	start := 3 + i*elemSize
	return a[start : start+elemSize]
}
