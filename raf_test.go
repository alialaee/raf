package raf

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestBuilderAndBlockBasic(t *testing.T) {
	b := NewBuilder(nil)

	keys := []KeyType{
		{"a_string", TypeString},
		{"b_false", TypeBool},
		{"b_true", TypeBool},
		{"c_float", TypeFloat64},
		{"d_int", TypeInt64},
	}

	b.AddKeys(keys...)

	// String
	b.AddString("hello world")
	b.AddBool(false)
	b.AddBool(true)
	b.AddFloat64(3.14159)
	b.AddInt64(-42)

	dst, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Verify Decoder Block
	block := NewBlock(dst)
	if !block.Valid() {
		t.Fatal("Block should be valid")
	}

	if block.NumPairs() != 5 {
		t.Fatalf("Expected 5 pairs, got %d", block.NumPairs())
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

	// Check non-existent
	_, ok = block.Get([]byte("z_missing"))
	if ok {
		t.Error("Expected missing key to return not ok")
	}
}

func TestArrayTypes(t *testing.T) {
	b := NewBuilder(nil)

	keys := []KeyType{
		{"a_strings", TypeArray},
		{"b_bools", TypeArray},
		{"c_floats", TypeArray},
		{"d_ints", TypeArray},
		{"e_empty", TypeArray},
	}

	b.AddKeys(keys...)

	// String array (dynamic)
	err := b.AddStringArray("hello", "world", "", "foo")
	if err != nil {
		t.Fatalf("AddStringArray: %v", err)
	}

	// Bool array
	err = b.AddBoolArray(true, false, true)
	if err != nil {
		t.Fatalf("AddBoolArray: %v", err)
	}

	// Float64 array
	err = b.AddFloat64Array(1.1, 2.2, 3.3)
	if err != nil {
		t.Fatalf("AddFloat64Array: %v", err)
	}

	// Int64 array
	err = b.AddInt64Array(10, -20, 30)
	if err != nil {
		t.Fatalf("AddInt64Array: %v", err)
	}

	// Empty string array
	err = b.AddStringArray()
	if err != nil {
		t.Fatalf("AddStringArray empty: %v", err)
	}

	// Build & validate block
	dst, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	block := NewBlock(dst)
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
		got := math.Float64frombits(binary.LittleEndian.Uint64(arr.At(i)))
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
		got := int64(binary.LittleEndian.Uint64(arr.At(i)))
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
	inner := NewBuilder(nil)
	inner.AddKeys(KeyType{
		Name: "x_val",
		Type: TypeString,
	}, KeyType{
		Name: "y_val",
		Type: TypeInt64,
	})

	inner.AddString("hello")
	inner.AddInt64(42)
	innerBytes, err := inner.Build()
	if err != nil {
		t.Fatalf("inner Build: %v", err)
	}

	// Build a deeply nested map: {"deep": "ok"}
	deep := NewBuilder(nil)
	deep.AddKeys(KeyType{
		Name: "deep",
		Type: TypeString,
	})

	deep.AddString("ok")
	deepBytes, err := deep.Build()
	if err != nil {
		t.Fatalf("deep Build: %v", err)
	}

	// Build empty map
	empty := NewBuilder(nil)
	emptyBytes, err := empty.Build()
	if err != nil {
		t.Fatalf("empty Build: %v", err)
	}

	// Build outer map with all three
	outer := NewBuilder(nil)
	outer.AddKeys(KeyType{
		Name: "a_map",
		Type: TypeMap,
	}, KeyType{
		Name: "b_nested",
		Type: TypeMap,
	}, KeyType{
		Name: "c_empty",
		Type: TypeMap,
	}, KeyType{
		Name: "d_plain",
		Type: TypeString,
	})

	outer.AddRaw(innerBytes)
	outer.AddRaw(deepBytes)
	outer.AddRaw(emptyBytes)
	outer.AddString("top-level")

	dst, err := outer.Build()
	if err != nil {
		t.Fatalf("outer Build: %v", err)
	}

	block := NewBlock(dst)
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
	innerBlock := val.Block()
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
	nestedBlock := val.Block()
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
	emptyBlock := val.Block()
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
