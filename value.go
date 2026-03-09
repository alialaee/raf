package raf

import (
	"encoding/binary"
	"math"
)

// Value represents a typed value read from a RAF block.
// It is valid only as long as the underlying block byte slice is valid.
type Value struct {
	Type Type
	Data []byte
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
	return int64(binary.BigEndian.Uint64(v.Data))
}

func (v Value) Float64() float64 {
	if v.Type != TypeFloat64 || len(v.Data) != 8 {
		return 0
	}
	return math.Float64frombits(binary.BigEndian.Uint64(v.Data))
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
		return nil
	}
	return Block(v.Data)
}

func (v Value) IsNull() bool {
	return v.Type == TypeNull
}
