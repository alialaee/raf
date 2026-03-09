package raf

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func TestBuilderAndBlockBasic(t *testing.T) {
	b := NewBuilder()

	// String
	err := b.AddString([]byte("a_string"), []byte("hello world"))
	if err != nil {
		t.Fatalf("AddString: %v", err)
	}

	// Bool false
	err = b.AddBool([]byte("b_false"), false)
	if err != nil {
		t.Fatalf("AddBool false: %v", err)
	}

	// Bool true
	err = b.AddBool([]byte("b_true"), true)
	if err != nil {
		t.Fatalf("AddBool true: %v", err)
	}

	// Float64
	err = b.AddFloat64([]byte("c_float"), 3.14159)
	if err != nil {
		t.Fatalf("AddFloat64: %v", err)
	}

	// Int64
	err = b.AddInt64([]byte("d_int"), -42)
	if err != nil {
		t.Fatalf("AddInt64: %v", err)
	}

	// Null
	err = b.AddNull([]byte("e_null"))
	if err != nil {
		t.Fatalf("AddNull: %v", err)
	}

	// Ensure build matches estimate
	est := b.EstimateSize()
	dst, err := b.Build(nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(dst) != est {
		t.Fatalf("Estimated size %d != built size %d", est, len(dst))
	}

	// Verify Decoder Block
	block := Block(dst)
	if !block.Valid() {
		t.Fatal("Block should be valid")
	}

	if block.NumPairs() != 6 {
		t.Fatalf("Expected 6 pairs, got %d", block.NumPairs())
	}

	// Check "a_string"
	val, ok := block.Get([]byte("a_string"))
	if !ok || val.Type != TypeString || val.String() != "hello world" {
		t.Errorf("Failed string check: %v %v %v", ok, val.Type, val.String())
	}

	// Check "b_false"
	val, ok = block.Get([]byte("b_false"))
	if !ok || val.Type != TypeBool || val.Bool() != false {
		t.Errorf("Failed bool false check: %v %v %v", ok, val.Type, val.Bool())
	}

	// Check "b_true"
	val, ok = block.Get([]byte("b_true"))
	if !ok || val.Type != TypeBool || val.Bool() != true {
		t.Errorf("Failed bool true check: %v %v %v", ok, val.Type, val.Bool())
	}

	// Check "c_float"
	val, ok = block.Get([]byte("c_float"))
	if !ok || val.Type != TypeFloat64 {
		t.Fatalf("Failed float check: %v %v %v", ok, val.Type, val.Data)
	}
	fval := val.Float64()
	if fval != 3.14159 {
		t.Errorf("Expected 3.14159, got %v", fval)
	}

	// Check "d_int"
	val, ok = block.Get([]byte("d_int"))
	if !ok || val.Type != TypeInt64 {
		t.Fatalf("Failed int check: %v %v %v", ok, val.Type, val.Data)
	}
	ival := val.Int64()
	if ival != -42 {
		t.Errorf("Expected -42, got %v", ival)
	}

	// Check "e_null"
	val, ok = block.Get([]byte("e_null"))
	if !ok || val.Type != TypeNull || len(val.Data) != 0 {
		t.Errorf("Failed null check: %v %v %v", ok, val.Type, val.Data)
	}

	// Check non-existent
	_, ok = block.Get([]byte("z_missing"))
	if ok {
		t.Error("Expected missing key to return not ok")
	}
}

func TestBuilderConstraints(t *testing.T) {
	b := NewBuilder()

	// Order check
	b.AddString([]byte("b"), []byte("value"))
	err := b.AddString([]byte("a"), []byte("value"))
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}

	// Duplicate check
	err = b.AddString([]byte("b"), []byte("another"))
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}

	// Zero-size check
	err = b.AddString([]byte(""), []byte("another"))
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}

	// Large key check
	err = b.AddString(make([]byte, maxKeySize+1), []byte("another"))
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}

	b.Reset()

	// Max pairs check (255)
	for i := range 255 {
		// generate 255 ordered keys using simple byte padding
		key := []byte{byte(i)}
		err := b.AddNull(key)
		if err != nil {
			t.Fatalf("Failed adding %d pair: %v", i, err)
		}
	}

	err = b.AddNull([]byte{255, 255})
	if !errors.Is(err, ErrTooManyPairs) {
		t.Errorf("Expected ErrTooManyPairs, got %v", err)
	}

	// Max Size check (> 64k)
	b.Reset()
	bigValue := make([]byte, math.MaxUint16)
	b.AddString([]byte("big"), bigValue)

	_, err = b.Build(nil)
	if !errors.Is(err, ErrBlockTooLarge) {
		t.Errorf("Expected ErrBlockTooLarge, got %v", err)
	}
}

func TestArrayTypes(t *testing.T) {
	b := NewBuilder()

	// String array (dynamic)
	err := b.AddStringArray([]byte("a_strings"), [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte(""),
		[]byte("foo"),
	})
	if err != nil {
		t.Fatalf("AddStringArray: %v", err)
	}

	// Bool array
	err = b.AddBoolArray([]byte("b_bools"), []bool{true, false, true})
	if err != nil {
		t.Fatalf("AddBoolArray: %v", err)
	}

	// Float64 array
	err = b.AddFloat64Array([]byte("c_floats"), []float64{1.1, 2.2, 3.3})
	if err != nil {
		t.Fatalf("AddFloat64Array: %v", err)
	}

	// Int64 array
	err = b.AddInt64Array([]byte("d_ints"), []int64{10, -20, 30})
	if err != nil {
		t.Fatalf("AddInt64Array: %v", err)
	}

	// Empty string array
	err = b.AddStringArray([]byte("e_empty"), nil)
	if err != nil {
		t.Fatalf("AddStringArray empty: %v", err)
	}

	// Build & validate block
	est := b.EstimateSize()
	dst, err := b.Build(nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(dst) != est {
		t.Fatalf("Estimated size %d != built size %d", est, len(dst))
	}

	block := Block(dst)
	if !block.Valid() {
		t.Fatal("Block should be valid")
	}
	if block.NumPairs() != 5 {
		t.Fatalf("Expected 5 pairs, got %d", block.NumPairs())
	}

	// === String array ===
	val, ok := block.Get([]byte("a_strings"))
	if !ok || val.Type != TypeArray {
		t.Fatalf("Failed string array lookup: ok=%v vt=%d", ok, val.Type)
	}
	arr := val.Array()
	if arr.ElemType() != TypeString {
		t.Fatalf("Expected TypeString elements, got %d", arr.ElemType())
	}
	if arr.Len() != 4 {
		t.Fatalf("Expected 4 string elements, got %d", arr.Len())
	}
	want := []string{"hello", "world", "", "foo"}
	for i, w := range want {
		if got := string(arr.At(i)); got != w {
			t.Errorf("String[%d]: got %q, want %q", i, got, w)
		}
	}

	// === Bool array ===
	val, ok = block.Get([]byte("b_bools"))
	if !ok || val.Type != TypeArray {
		t.Fatalf("Failed bool array lookup")
	}
	arr = val.Array()
	if arr.ElemType() != TypeBool || arr.Len() != 3 {
		t.Fatalf("Bool array header: type=%d len=%d", arr.ElemType(), arr.Len())
	}
	wantBools := []bool{true, false, true}
	for i, w := range wantBools {
		got := arr.At(i)[0] != 0
		if got != w {
			t.Errorf("Bool[%d]: got %v, want %v", i, got, w)
		}
	}

	// === Float64 array ===
	val, ok = block.Get([]byte("c_floats"))
	if !ok || val.Type != TypeArray {
		t.Fatalf("Failed float64 array lookup")
	}
	arr = val.Array()
	if arr.ElemType() != TypeFloat64 || arr.Len() != 3 {
		t.Fatalf("Float64 array header: type=%d len=%d", arr.ElemType(), arr.Len())
	}
	wantFloats := []float64{1.1, 2.2, 3.3}
	for i, w := range wantFloats {
		got := math.Float64frombits(binary.BigEndian.Uint64(arr.At(i)))
		if got != w {
			t.Errorf("Float64[%d]: got %v, want %v", i, got, w)
		}
	}

	// === Int64 array ===
	val, ok = block.Get([]byte("d_ints"))
	if !ok || val.Type != TypeArray {
		t.Fatalf("Failed int64 array lookup")
	}
	arr = val.Array()
	if arr.ElemType() != TypeInt64 || arr.Len() != 3 {
		t.Fatalf("Int64 array header: type=%d len=%d", arr.ElemType(), arr.Len())
	}
	wantInts := []int64{10, -20, 30}
	for i, w := range wantInts {
		got := int64(binary.BigEndian.Uint64(arr.At(i)))
		if got != w {
			t.Errorf("Int64[%d]: got %v, want %v", i, got, w)
		}
	}

	// === Empty array ===
	val, ok = block.Get([]byte("e_empty"))
	if !ok || val.Type != TypeArray {
		t.Fatalf("Failed empty array lookup")
	}
	arr = val.Array()
	if arr.ElemType() != TypeString || arr.Len() != 0 {
		t.Fatalf("Empty array: type=%d len=%d", arr.ElemType(), arr.Len())
	}
}

func TestMapType(t *testing.T) {
	// Build inner map: {"x_val": "hello", "y_val": int64(42)}
	inner := NewBuilder()
	inner.AddString([]byte("x_val"), []byte("hello"))
	inner.AddInt64([]byte("y_val"), 42)
	innerBytes, err := inner.Build(nil)
	if err != nil {
		t.Fatalf("inner Build: %v", err)
	}

	// Build a deeply nested map: {"deep": "ok"}
	deep := NewBuilder()
	deep.AddString([]byte("deep"), []byte("ok"))
	deepBytes, err := deep.Build(nil)
	if err != nil {
		t.Fatalf("deep Build: %v", err)
	}

	// Build empty map
	empty := NewBuilder()
	emptyBytes, err := empty.Build(nil)
	if err != nil {
		t.Fatalf("empty Build: %v", err)
	}

	// Build outer map with all three
	outer := NewBuilder()
	outer.AddMap([]byte("a_map"), innerBytes)
	outer.AddMap([]byte("b_nested"), deepBytes)
	outer.AddMap([]byte("c_empty"), emptyBytes)
	outer.AddString([]byte("d_plain"), []byte("top-level"))

	est := outer.EstimateSize()
	dst, err := outer.Build(nil)
	if err != nil {
		t.Fatalf("outer Build: %v", err)
	}
	if len(dst) != est {
		t.Fatalf("Estimated size %d != built size %d", est, len(dst))
	}

	block := Block(dst)
	if !block.Valid() {
		t.Fatal("Outer block should be valid")
	}
	if block.NumPairs() != 4 {
		t.Fatalf("Expected 4 pairs, got %d", block.NumPairs())
	}

	// === Read inner map ===
	val, ok := block.Get([]byte("a_map"))
	if !ok || val.Type != TypeMap {
		t.Fatalf("Failed map lookup: ok=%v vt=%d", ok, val.Type)
	}
	innerBlock := val.Map()
	if !innerBlock.Valid() {
		t.Fatal("Inner block should be valid")
	}
	if innerBlock.NumPairs() != 2 {
		t.Fatalf("Inner: expected 2 pairs, got %d", innerBlock.NumPairs())
	}

	ival, iok := innerBlock.Get([]byte("x_val"))
	if !iok || ival.Type != TypeString || ival.String() != "hello" {
		t.Errorf("Inner x_val: ok=%v vt=%d val=%q", iok, ival.Type, ival.String())
	}

	ival, iok = innerBlock.Get([]byte("y_val"))
	if !iok || ival.Type != TypeInt64 {
		t.Fatalf("Inner y_val: ok=%v vt=%d", iok, ival.Type)
	}
	ival64 := ival.Int64()
	if ival64 != 42 {
		t.Errorf("Inner y_val: expected 42, got %d", ival64)
	}

	// === Read nested map ===
	val, ok = block.Get([]byte("b_nested"))
	if !ok || val.Type != TypeMap {
		t.Fatalf("Failed nested map lookup")
	}
	nestedBlock := val.Map()
	if !nestedBlock.Valid() {
		t.Fatal("Nested block should be valid")
	}
	nval, nok := nestedBlock.Get([]byte("deep"))
	if !nok || nval.Type != TypeString || nval.String() != "ok" {
		t.Errorf("Nested deep: ok=%v vt=%d val=%q", nok, nval.Type, nval.String())
	}

	// === Read empty map ===
	val, ok = block.Get([]byte("c_empty"))
	if !ok || val.Type != TypeMap {
		t.Fatalf("Failed empty map lookup")
	}
	emptyBlock := val.Map()
	if !emptyBlock.Valid() {
		t.Fatal("Empty block should be valid")
	}
	if emptyBlock.NumPairs() != 0 {
		t.Fatalf("Empty map: expected 0 pairs, got %d", emptyBlock.NumPairs())
	}

	// === Other fields ===
	val, ok = block.Get([]byte("d_plain"))
	if !ok || val.Type != TypeString || val.String() != "top-level" {
		t.Errorf("Plain value: ok=%v vt=%d val=%q", ok, val.Type, val.String())
	}
}

func TestValueBytes(t *testing.T) {
	b := NewBuilder()
	b.AddStringString([]byte("my_str"), "hello bytes")

	valBuf := make([]byte, 0, 32)
	dst, _ := b.Build(nil)
	block := Block(dst)

	val, ok := block.Get([]byte("my_str"))
	if !ok {
		t.Fatal("Failed to get my_str")
	}

	// Test Value.Bytes
	bStr := val.Bytes(valBuf)
	if string(bStr) != "hello bytes" {
		t.Errorf("Expected 'hello bytes', got %q", bStr)
	}

	// Test Value.Bytes on wrong type
	b.Reset()
	b.AddInt64([]byte("my_int"), 123)
	dst, _ = b.Build(nil)
	block = Block(dst)
	val, _ = block.Get([]byte("my_int"))
	if val.Bytes(valBuf) != nil {
		t.Error("Expected nil when calling Bytes on non-string value")
	}
}

func TestArrayAtHelpers(t *testing.T) {
	b := NewBuilder()

	err := b.AddBoolArray([]byte("a_bool"), []bool{true, false})
	if err != nil {
		t.Fatal(err)
	}
	err = b.AddFloat64Array([]byte("b_float"), []float64{1.1, 2.2})
	if err != nil {
		t.Fatal(err)
	}
	err = b.AddInt64Array([]byte("c_int"), []int64{10, 20})
	if err != nil {
		t.Fatal(err)
	}

	// Use the new AddStringStringArray helper
	err = b.AddStringStringArray([]byte("d_str"), []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}

	dst, err := b.Build(nil)
	if err != nil {
		t.Fatal(err)
	}
	block := Block(dst)

	valBuf := make([]byte, 0, 16)

	// Bool array helper
	arrVal, ok := block.Get([]byte("a_bool"))
	if !ok {
		t.Fatal("missing a_bool")
	}
	arr := arrVal.Array()
	if arr.AtBool(0) != true || arr.AtBool(1) != false {
		t.Errorf("AtBool expected true, false. got %v, %v", arr.AtBool(0), arr.AtBool(1))
	}
	// wrong type fallbacks
	if arr.AtString(0, valBuf) != nil {
		t.Errorf("AtString on Bool array should return nil")
	}
	if arr.AtFloat64(0) != 0 {
		t.Errorf("AtFloat64 on Int64 array should return 0")
	}
	if arr.AtInt64(0) != 0 {
		t.Errorf("AtInt64 on String array should return 0")
	}

	// Float64 array helper
	arrVal, ok = block.Get([]byte("b_float"))
	if !ok {
		t.Fatal("missing b_float")
	}
	arr = arrVal.Array()
	if arr.AtFloat64(0) != 1.1 {
		t.Errorf("AtFloat64(0) expected 1.1, got %f", arr.AtFloat64(0))
	}

	// Int64 array helper
	arrVal, ok = block.Get([]byte("c_int"))
	if !ok {
		t.Fatal("missing c_int")
	}
	arr = arrVal.Array()
	if arr.AtInt64(1) != 20 {
		t.Errorf("AtInt64(1) expected 20, got %d", arr.AtInt64(1))
	}

	// String array helper
	arrVal, ok = block.Get([]byte("d_str"))
	if !ok {
		t.Fatal("missing d_str")
	}
	arr = arrVal.Array()
	if string(arr.AtString(0, valBuf)) != "a" {
		t.Errorf("AtString(0) expected 'a', got %q", arr.AtString(0, valBuf))
	}
}
