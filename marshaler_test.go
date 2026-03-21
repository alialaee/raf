package raf_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/alialaee/raf"
)

type ComplexStruct struct {
	String string `raf:"string"`
	Bool   bool   `raf:"bool"`

	Int   int   `raf:"int"`
	Int8  int8  `raf:"int8"`
	Int16 int16 `raf:"int16"`
	Int32 int32 `raf:"int32"`
	Int64 int64 `raf:"int64"`

	Uint   uint   `raf:"uint"`
	Uint8  uint8  `raf:"uint8"`
	Uint16 uint16 `raf:"uint16"`
	Uint32 uint32 `raf:"uint32"`
	Uint64 uint64 `raf:"uint64"`

	Float32 float32 `raf:"float32"`
	Float64 float64 `raf:"float64"`

	IntPointer    *int    `raf:"int_pointer"`
	StringPointer *string `raf:"string_pointer"`

	Strings []string `raf:"strings"`
	Bools   []bool   `raf:"bools"`

	Ints   []int   `raf:"ints"`
	Int8s  []int8  `raf:"int8s"`
	Int16s []int16 `raf:"int16s"`
	Int32s []int32 `raf:"int32s"`
	Int64s []int64 `raf:"int64s"`

	Uints   []uint   `raf:"uints"`
	Uint8s  []uint8  `raf:"uint8s"`
	Uint16s []uint16 `raf:"uint16s"`
	Uint32s []uint32 `raf:"uint32s"`
	Uint64s []uint64 `raf:"uint64s"`

	Float32s []float32 `raf:"floats32"`
	Float64s []float64 `raf:"floats64"`

	InnerStruct InnerStruct `raf:"inner_struct"`

	Pairs []Pair `raf:"pairs"`

	PairPointer *Pair `raf:"pair_pointer"`
}

type InnerStruct struct {
	Field1 string `raf:"field1"`
	Field2 int    `raf:"field2"`

	InnerInner InnerInner `raf:"inner_inner"`
}

type InnerInner struct {
	Strings []string `raf:"strings"`
	Ints    []int    `raf:"ints"`
}

type Pair struct {
	A string `raf:"a"`
	B string `raf:"b"`
}

func makeComplexStruct() ComplexStruct {
	return ComplexStruct{
		String: "hello",
		Bool:   true,

		Int:   -42,
		Int8:  -8,
		Int16: -16,
		Int32: -32,
		Int64: -64,

		Uint:   42,
		Uint8:  8,
		Uint16: 16,
		Uint32: 32,
		Uint64: 64,

		Float32: 3.14,
		Float64: 2.71828,

		IntPointer:    new(42),
		StringPointer: new("hello pointer"),

		Strings: []string{"foo", "bar", "baz"},
		Bools:   []bool{true, false, true},

		Ints:   []int{-1000, -101, 0, 200, 500},
		Int8s:  []int8{-100, -50, 0, 50, 100},
		Int16s: []int16{-1000, -500, 0, 500, 1000},
		Int32s: []int32{-100000, -50000, 0, 50000, 100000},
		Int64s: []int64{-10000000000, -5000000000, 0, 5000000000, 10000000000},

		Uints:   []uint{1000, 101, 0, 200, 500},
		Uint8s:  []uint8{100, 50, 0, 50, 100},
		Uint16s: []uint16{1000, 500, 0, 500, 1000},
		Uint32s: []uint32{100000, 50000, 0, 50000, 100000},
		Uint64s: []uint64{10000000000, 5000000000, 0, 5000000000, 10000000000},

		Float32s: []float32{3.14, -2.71},
		Float64s: []float64{2.71828, -3.14},

		InnerStruct: InnerStruct{
			Field1: "inner",
			Field2: 123,
			InnerInner: InnerInner{
				Strings: []string{"innerfoo", "innerbar"},
				Ints:    []int{1, 2, 3},
			},
		},

		Pairs: []Pair{
			{A: "first", B: "pair"},
			{A: "second", B: "pair"},
		},

		PairPointer: &Pair{A: "pointer", B: "pair"},
	}
}

func testMarshalUnmarshal_WithSelf[T any](t *testing.T, original T) {
	t.Helper()
	testMarshalUnmarshal(t, original, original)
}

func testMarshalUnmarshal[T any, V any](t *testing.T, original T, expected V) {
	t.Helper()
	data, err := raf.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded V
	if err := raf.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !reflect.DeepEqual(expected, decoded) {
		t.Fatalf("decoded value does not match expected.\nExpected: %+v\nDecoded: %+v", expected, decoded)
	}
}

func TestComplexStruct(t *testing.T) {
	original := makeComplexStruct()
	testMarshalUnmarshal_WithSelf(t, original)
}

func TestNilValues(t *testing.T) {
	type StructWithPointers struct {
		IntPointer     *int      `raf:"int_pointer"`
		StringPointer  *string   `raf:"string_pointer"`
		Strings        []string  `raf:"strings"`
		StringsPointer *[]string `raf:"strings_pointer"`
	}

	original := StructWithPointers{
		IntPointer:     nil,
		StringPointer:  nil,
		Strings:        nil,
		StringsPointer: nil,
	}

	testMarshalUnmarshal_WithSelf(t, original)
}

func TestEmptySlices(t *testing.T) {
	type StructWithSlices struct {
		Strings []string `raf:"strings"`
		Ints    []int    `raf:"ints"`
		Int     int      `raf:"int"`
	}

	original := StructWithSlices{
		Strings: []string{},
		Ints:    []int{},
		Int:     0,
	}

	testMarshalUnmarshal_WithSelf(t, original)
}

func TestNilToEmpty(t *testing.T) {
	type StructWithPointer struct {
		Int *int `raf:"int"`
	}

	type StructWithValue struct {
		Int int `raf:"int"`
	}

	original := StructWithPointer{
		Int: nil,
	}

	expected := StructWithValue{
		Int: 0,
	}

	testMarshalUnmarshal(t, original, expected)
}

func TestEmptyToNil(t *testing.T) {
	type StructWithPointer struct {
		Int *int `raf:"int"`
	}

	type StructWithValue struct {
		Int int `raf:"int"`
	}

	original := StructWithValue{
		Int: 0,
	}

	expected := StructWithPointer{
		Int: new(0),
	}

	testMarshalUnmarshal(t, original, expected)
}

func TestUnmarshalMap(t *testing.T) {
	complexStruct := makeComplexStruct()
	data, err := raf.Marshal(complexStruct)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := map[string]any{
		"string": "hello",
		"bool":   true,

		"int":   int64(-42),
		"int8":  int64(-8),
		"int16": int64(-16),
		"int32": int64(-32),
		"int64": int64(-64),

		"uint":   int64(42),
		"uint8":  int64(8),
		"uint16": int64(16),
		"uint32": int64(32),
		"uint64": int64(64),

		"float32": float64(float32(3.14)),
		"float64": float64(2.71828),

		"int_pointer":    int64(42),
		"string_pointer": "hello pointer",

		"strings": []string{"foo", "bar", "baz"},
		"bools":   []bool{true, false, true}, // TODO fix

		"ints":   []int64{-1000, -101, 0, 200, 500},
		"int8s":  []int64{-100, -50, 0, 50, 100},
		"int16s": []int64{-1000, -500, 0, 500, 1000},
		"int32s": []int64{-100000, -50000, 0, 50000, 100000},
		"int64s": []int64{-10000000000, -5000000000, 0, 5000000000, 10000000000},

		"uints":   []int64{1000, 101, 0, 200, 500}, // TODO fix
		"uint8s":  []int64{100, 50, 0, 50, 100},
		"uint16s": []int64{1000, 500, 0, 500, 1000},
		"uint32s": []int64{100000, 50000, 0, 50000, 100000},
		"uint64s": []int64{10000000000, 5000000000, 0, 5000000000, 10000000000},

		"floats32": []float64{float64(float32(3.14)), float64(float32(-2.71))},
		"floats64": []float64{2.71828, -3.14},

		"inner_struct": map[string]any{
			"field1": "inner",
			"field2": int64(123),
			"inner_inner": map[string]any{
				"strings": []string{"innerfoo", "innerbar"},
				"ints":    []int64{1, 2, 3},
			},
		},

		"pairs": []map[string]any{
			{"a": "first", "b": "pair"},
			{"a": "second", "b": "pair"},
		},

		"pair_pointer": map[string]any{"a": "pointer", "b": "pair"},
	}

	var got map[string]any
	if err := raf.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for k, v := range expected {
		gotVal, ok := got[k]
		if !ok {
			t.Fatalf("missing key in result: %s", k)
		}

		if !reflect.DeepEqual(gotVal, v) {
			t.Fatalf("unexpected value for key %q: got (%T)%+v, want (%T)%+v", k, gotVal, gotVal, v, v)
		}
	}
}

func TestMarshalAndUnmarshalMap(t *testing.T) {
	data := map[string]any{
		"string": "hello",
		"bool":   true,

		"int":   int64(-42),
		"int8":  int64(-8),
		"int16": int64(-16),
		"int32": int64(-32),
		"int64": int64(-64),

		"uint":   int64(42),
		"uint8":  int64(8),
		"uint16": int64(16),
		"uint32": int64(32),
		"uint64": int64(64),

		"float32": float64(float32(3.14)),
		"float64": float64(2.71828),

		"int_pointer":    int64(42),
		"string_pointer": "hello pointer",

		"strings": []string{"foo", "bar", "baz"},
		"bools":   []bool{true, false, true}, // TODO fix

		"ints":   []int64{-1000, -101, 0, 200, 500},
		"int8s":  []int64{-100, -50, 0, 50, 100},
		"int16s": []int64{-1000, -500, 0, 500, 1000},
		"int32s": []int64{-100000, -50000, 0, 50000, 100000},
		"int64s": []int64{-10000000000, -5000000000, 0, 5000000000, 10000000000},

		"uints":   []int64{1000, 101, 0, 200, 500}, // TODO fix
		"uint8s":  []int64{100, 50, 0, 50, 100},
		"uint16s": []int64{1000, 500, 0, 500, 1000},
		"uint32s": []int64{100000, 50000, 0, 50000, 100000},
		"uint64s": []int64{10000000000, 5000000000, 0, 5000000000, 10000000000},

		"floats32": []float64{float64(float32(3.14)), float64(float32(-2.71))},
		"floats64": []float64{2.71828, -3.14},

		"inner_struct": map[string]any{
			"field1": "inner",
			"field2": int64(123),
			"inner_inner": map[string]any{
				"strings": []string{"innerfoo", "innerbar"},
				"ints":    []int64{1, 2, 3},
			},
		},

		"pairs": []map[string]any{
			{"a": "first", "b": "pair"},
			{"a": "second", "b": "pair"},
		},

		"pair_pointer": map[string]any{"a": "pointer", "b": "pair"},
	}

	marshaled, err := raf.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled map[string]any
	if err := raf.Unmarshal(marshaled, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for k, v := range data {
		gotVal, ok := unmarshaled[k]
		if !ok {
			t.Fatalf("missing key in result: %s", k)
		}

		if !reflect.DeepEqual(gotVal, v) {
			t.Fatalf("unexpected value for key %q: got (%T)%+v, want (%T)%+v", k, gotVal, gotVal, v, v)
		}
	}
}

func TestUnmarshal_Failed_InvalidData(t *testing.T) {
	type A struct {
		Num int `raf:"num"`
	}

	data := []byte{0x01, 0x00, 0x01, 0x01, 0x00, 0x00}

	var a A
	err := raf.Unmarshal(data, &a)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	expectedErr := "invalid RAF data"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expectedErr)
	}
}

func TestUnmarshal_Failed_TypeMismatch(t *testing.T) {
	type A struct {
		Num int `raf:"num"`
	}

	type B struct {
		Num string `raf:"num"`
	}

	data, err := raf.Marshal(A{Num: 42})
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var b B
	err = raf.Unmarshal(data, &b)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	expectedErr := "type mismatch for key num: expected int64, got string"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expectedErr)
	}
}

func TestMinMaxValues(t *testing.T) {
	type MinInts struct {
		Int8  int8  `raf:"int8"`
		Int16 int16 `raf:"int16"`
		Int32 int32 `raf:"int32"`
		Int64 int64 `raf:"int64"`
	}

	testMarshalUnmarshal_WithSelf(t, MinInts{
		Int8:  math.MinInt8,
		Int16: math.MinInt16,
		Int32: math.MinInt32,
		Int64: math.MinInt64,
	})

	type MaxInts struct {
		Int8   int8   `raf:"int8"`
		Int16  int16  `raf:"int16"`
		Int32  int32  `raf:"int32"`
		Int64  int64  `raf:"int64"`
		Uint8  uint8  `raf:"uint8"`
		Uint16 uint16 `raf:"uint16"`
		Uint32 uint32 `raf:"uint32"`
		Uint64 uint64 `raf:"uint64"`
	}

	testMarshalUnmarshal_WithSelf(t, MaxInts{
		Int8:   math.MaxInt8,
		Int16:  math.MaxInt16,
		Int32:  math.MaxInt32,
		Int64:  math.MaxInt64,
		Uint8:  math.MaxUint8,
		Uint16: math.MaxUint16,
		Uint32: math.MaxUint32,
		Uint64: math.MaxUint64,
	})
}

func TestWithoutTags(t *testing.T) {
	type NoTags struct {
		Num  int
		Str  string
		Bool bool
		Num2 int
	}

	type WithTags struct {
		Num  int    `raf:"num"`
		Str  string `raf:"str"`
		Bool bool   `raf:"bool"`
		Num2 int    `raf:"-"`
	}

	testMarshalUnmarshal(t, WithTags{
		Num:  42,
		Str:  "Hello",
		Bool: true,
		Num2: 100,
	}, NoTags{
		Num:  42,
		Str:  "Hello",
		Bool: true,
		Num2: 0,
	})
}

func TestUnexportedFields(t *testing.T) {
	type WithUnexported struct {
		Exported   string `raf:"exported"`
		unexported string `raf:"unexported"`
	}

	testMarshalUnmarshal(t, WithUnexported{
		Exported:   "This is exported",
		unexported: "This is unexported",
	},
		WithUnexported{
			Exported:   "This is exported",
			unexported: "",
		},
	)
}

func TestMissingFields(t *testing.T) {
	type A struct {
		Num1  int     `raf:"num1"`
		Str   string  `raf:"str"`
		Num2  int     `raf:"num2"`
		Bool  bool    `raf:"bool"`
		Float float64 `raf:"float"`
	}

	type B struct {
		Num1 int  `raf:"num1"`
		Num2 int  `raf:"num2"`
		Bool bool `raf:"bool"`
	}

	testMarshalUnmarshal(t, A{
		Num1:  42,
		Str:   "Hello",
		Num2:  100,
		Bool:  true,
		Float: 3.14,
	}, B{
		Num1: 42,
		Num2: 100,
		Bool: true,
	})

	testMarshalUnmarshal(t, B{
		Num1: 42,
		Num2: 100,
		Bool: true,
	}, A{
		Num1:  42,
		Str:   "",
		Num2:  100,
		Bool:  true,
		Float: 0,
	})
}
