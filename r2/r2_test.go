package r2

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/alialaee/raf"
)

type Contact struct {
	Email string `raf:"email"`
	Phone string `raf:"phone"`
}

type Player struct {
	ID     int     `raf:"id"`
	Name   string  `raf:"name"`
	Age    *int    `raf:"age"`
	Alive  bool    `raf:"alive"`
	Happy  bool    `raf:"happy"`
	Weight float64 `raf:"weight"`
	Mother string  `raf:"mother"`

	Contacts []Contact `raf:"contact"`

	Friends   []string `raf:"friends"`
	FrindsAge []int    `raf:"friends_age"`
}

var pToMarshal = Player{
	ID:     12345,
	Name:   "Alice",
	Age:    new(30),
	Alive:  true,
	Happy:  false,
	Weight: 150.5,
	Mother: "Eve",
	Contacts: []Contact{
		{
			Email: "alice@alice.com",
			Phone: "123-456-7890",
		},
		{
			Email: "bob@bob.com",
			Phone: "987-654-3210",
		},
	},
	Friends:   []string{"Bob", "Charlie"},
	FrindsAge: []int{25, 28},
}

func BenchmarkOpCodeUnmarshal(b *testing.B) {
	data, err := raf.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	dec := NewUnmarshaler()

	p := Player{}
	for b.Loop() {
		if err := dec.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	if !reflect.DeepEqual(p, pToMarshal) {
		b.Fatalf("unexpected result: got %+v, want %+v", p, pToMarshal)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	data, err := raf.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	var p Player
	b.ResetTimer()
	for b.Loop() {
		if err := raf.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	data, err := json.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	var p Player
	b.ResetTimer()
	for b.Loop() {
		if err := json.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}

func testUnmarshal[T any, V any](t *testing.T, given T, expected V) {
	t.Helper()

	data, err := raf.Marshal(given)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got V
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected result: got %+v, want %+v", got, expected)
	}
}

func testUnmarshalSame[T any](t *testing.T, v T) {
	t.Helper()

	data, err := raf.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got T
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !reflect.DeepEqual(got, v) {
		t.Fatalf("unexpected result: got %+v, want %+v", got, v)
	}
}

type FullPrimitive struct {
	Str       string  `raf:"str"`
	Int8      int8    `raf:"int8"`
	Int16     int16   `raf:"int16"`
	Int32     int32   `raf:"int32"`
	Int64     int64   `raf:"int64"`
	Int       int     `raf:"int"`
	Uint8     uint8   `raf:"uint8"`
	Uint16    uint16  `raf:"uint16"`
	Uint32    uint32  `raf:"uint32"`
	Uint64    uint64  `raf:"uint64"`
	Uint      uint    `raf:"uint"`
	Float32   float32 `raf:"float32"`
	Float64   float64 `raf:"float64"`
	Bool      bool    `raf:"bool"`
	Null      *string `raf:"null"`
	PtrStr    *string `raf:"ptr_str"`
	PtrInt    *int    `raf:"ptr_int"`
	PtrNull   *string `raf:"ptr_null"`
	EmptyStr  string  `raf:"empty_str"`
	EmptyPtr  *string `raf:"empty_ptr"`
	EmptyInt  *int    `raf:"empty_int"`
	EmptyBool *bool   `raf:"empty_bool"`

	Ints    []int    `raf:"ints"`
	Ints8   []int8   `raf:"ints8"`
	Ints16  []int16  `raf:"ints16"`
	Ints32  []int32  `raf:"ints32"`
	Ints64  []int64  `raf:"ints64"`
	Uints   []uint   `raf:"uints"`
	Uints8  []uint8  `raf:"uints8"`
	Uints16 []uint16 `raf:"uints16"`
	Uints32 []uint32 `raf:"uints32"`
	Uints64 []uint64 `raf:"uints64"`

	Floats32 []float32 `raf:"floats32"`
	Floats64 []float64 `raf:"floats64"`
	Bools    []bool    `raf:"bools"`
	// IntPointers []*int    `raf:"int_pointers"`

	Strs []string `raf:"strings"`
}

func getFullPrimitive() FullPrimitive {
	return FullPrimitive{
		Str:       "Hello, RAF!",
		Int8:      -8,
		Int16:     -16,
		Int32:     -32,
		Int64:     -64,
		Int:       -12345,
		Uint8:     8,
		Uint16:    16,
		Uint32:    32,
		Uint64:    64,
		Uint:      12345,
		Float32:   3.14,
		Float64:   2.71828,
		Bool:      true,
		Null:      nil,
		PtrStr:    new(string("Pointer to string")),
		PtrInt:    new(int(42)),
		PtrNull:   nil,
		EmptyStr:  "",
		EmptyPtr:  new(string("")),
		EmptyInt:  new(int(0)),
		EmptyBool: new(bool(false)),

		Ints:   []int{-1, 0, 1},
		Ints8:  []int8{-8, 0, 8},
		Ints16: []int16{-16, 0, 16},
		Ints32: []int32{-32, 0, 32},
		Ints64: []int64{-64, 0, 64},

		Uints:   []uint{123, 123, 5, 66, 8, 9, 5},
		Uints8:  []uint8{234, 16},
		Uints16: []uint16{5, 5, 5, 5, 6},
		Uints32: []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1},
		Uints64: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9},

		Floats32: []float32{3.14, -2.71},
		Floats64: []float64{2.71828, -3.14},
		Bools:    []bool{true, false, true},

		Strs: []string{"Hello", "RAF", "Test"},
	}
}

func TestUnmarshal_Success(t *testing.T) {
	t.Run("All primitive types", func(t *testing.T) {
		testUnmarshal(t, getFullPrimitive(), getFullPrimitive())
	})

	t.Run("Empty struct", func(t *testing.T) {
		type Empty struct{}
		testUnmarshal(t, Empty{}, Empty{})
	})

	t.Run("Struct with only empty values", func(t *testing.T) {
		type OnlyEmpty struct {
			EmptyStr  string  `raf:"empty_str"`
			EmptyPtr  *string `raf:"empty_ptr"`
			EmptyInt  *int    `raf:"empty_int"`
			EmptyBool *bool   `raf:"empty_bool"`
		}
		testUnmarshalSame(t, OnlyEmpty{
			EmptyStr:  "",
			EmptyPtr:  new(string("")),
			EmptyInt:  new(int(0)),
			EmptyBool: new(bool(false)),
		})
	})

	t.Run("Struct with nil pointer fields", func(t *testing.T) {
		type WithNil struct {
			Str    string  `raf:"str"`
			PtrStr *string `raf:"ptr_str"`
			PtrInt *int    `raf:"ptr_int"`
		}
		testUnmarshal(t, WithNil{
			Str:    "Hello",
			PtrStr: nil,
			PtrInt: nil,
		}, WithNil{
			Str:    "Hello",
			PtrStr: nil,
			PtrInt: nil,
		})
	})

	t.Run("Nested struct", func(t *testing.T) {
		type Nested struct {
			Message       string `raf:"message"`
			FullPrimitive `raf:"full_primitive"`
		}

		testUnmarshalSame(t, Nested{
			Message:       "Nested struct test",
			FullPrimitive: getFullPrimitive(),
		})
	})

	t.Run("Struct with array fields", func(t *testing.T) {
		type WithArrays struct {
			Strs        []string  `raf:"strs"`
			Ints        []int     `raf:"ints"`
			Ints8       []int8    `raf:"ints8"`
			Ints16      []int16   `raf:"ints16"`
			Ints32      []int32   `raf:"ints32"`
			Ints64      []int64   `raf:"ints64"`
			Uints       []uint    `raf:"uints"`
			Uints8      []uint8   `raf:"uints8"`
			Uints16     []uint16  `raf:"uints16"`
			Uints32     []uint32  `raf:"uints32"`
			Uints64     []uint64  `raf:"uints64"`
			Float32     []float32 `raf:"float32s"`
			Float64     []float64 `raf:"float64s"`
			Bools       []bool    `raf:"bools"`
			IntPointers []*int    `raf:"int_pointers"`
		}

		testUnmarshalSame(t, WithArrays{
			Strs:    []string{"Hello", "RAF", "Test"},
			Ints:    []int{-1, 0, 1},
			Ints8:   []int8{-8, 0, 8},
			Ints16:  []int16{-16, 0, 16},
			Ints32:  []int32{-32, 0, 32},
			Ints64:  []int64{-64, 0, 64},
			Uints:   []uint{0, 1, 12345},
			Uints8:  []uint8{0, 8},
			Uints16: []uint16{0, 16},
			Uints32: []uint32{0, 32},
			Uints64: []uint64{0, 64},
			Float32: []float32{3.14, -2.71},
			Float64: []float64{2.71828, -3.14},
			Bools:   []bool{true, false, true},
			// IntPointers: []*int{nil, new(int(0)), new(int(1))}, // TODO
		})
	})

	t.Run("Struct with array of structs", func(t *testing.T) {
		type Inner struct {
			A int    `raf:"a"`
			B string `raf:"b"`
		}

		type WithArrayOfStructs struct {
			Array []Inner `raf:"array"`
		}

		testUnmarshalSame(t, WithArrayOfStructs{
			Array: []Inner{
				{A: 1, B: "One"},
				{A: 2, B: "Two"},
				{A: 3, B: "Three"},
			},
		})
	})

	// t.Run("Struct with array of arrays", func(t *testing.T) {
	// 	type WithArrayOfArrays struct {
	// 		ArrayOfArrays [][]string `raf:"array_of_arrays"`
	// 	}

	// 	testUnmarshalSame(t, WithArrayOfArrays{
	// 		ArrayOfArrays: [][]string{
	// 			{"Hello", "World"},
	// 			{"RAF", "Test"},
	// 		},
	// 	})
	// })

	t.Run("Struct with empty array fields", func(t *testing.T) {
		type WithEmptyArrays struct {
			Strs    []string  `raf:"strs"`
			Ints    []int     `raf:"ints"`
			Float64 []float64 `raf:"float64s"`
		}

		testUnmarshalSame(t, WithEmptyArrays{
			Strs:    []string{},
			Ints:    []int{},
			Float64: []float64{},
		})
	})

	t.Run("Struct with nil slice fields", func(t *testing.T) {
		type WithNilSlices struct {
			Strs    []string  `raf:"strs"`
			Ints    []int     `raf:"ints"`
			Float64 []float64 `raf:"float64s"`
		}

		testUnmarshalSame(t, WithNilSlices{
			Strs:    nil,
			Ints:    nil,
			Float64: nil,
		})
	})

	t.Run("Struct with unexported fields", func(t *testing.T) {
		type WithUnexported struct {
			Exported   string `raf:"exported"`
			unexported string `raf:"unexported"`
		}

		testUnmarshal(t, WithUnexported{
			Exported:   "This is exported",
			unexported: "This is unexported",
		},
			WithUnexported{
				Exported:   "This is exported",
				unexported: "",
			},
		)
	})

	t.Run("Struct with missing fields in RAF data", func(t *testing.T) {
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

		testUnmarshal(t, A{
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

		testUnmarshal(t, B{
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
	})

	t.Run("Struct with '-' fields in RAF data", func(t *testing.T) {
		type WithIgnored struct {
			Num1 int  `raf:"num1"`
			Num2 int  `raf:"-"`
			Bool bool `raf:"bool"`
		}

		testUnmarshal(t, WithIgnored{
			Num1: 42,
			Num2: 100,
			Bool: true,
		}, WithIgnored{
			Num1: 42,
			Num2: 0,
			Bool: true,
		})
	})

	t.Run("Struct without RAF tags", func(t *testing.T) {
		type NoTags struct {
			Num  int
			Str  string
			Bool bool
		}

		type WithTags struct {
			Num  int    `raf:"num"`
			Str  string `raf:"str"`
			Bool bool   `raf:"bool"`
		}

		testUnmarshal(t, WithTags{
			Num:  42,
			Str:  "Hello",
			Bool: true,
		}, NoTags{
			Num:  42,
			Str:  "Hello",
			Bool: true,
		})
	})

	t.Run("Maximum values for integer types", func(t *testing.T) {
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

		testUnmarshalSame(t, MaxInts{
			Int8:   math.MaxInt8,
			Int16:  math.MaxInt16,
			Int32:  math.MaxInt32,
			Int64:  math.MaxInt64,
			Uint8:  math.MaxUint8,
			Uint16: math.MaxUint16,
			Uint32: math.MaxUint32,
			Uint64: math.MaxUint64,
		})
	})

	t.Run("Minimum values for integer types", func(t *testing.T) {
		type MinInts struct {
			Int8  int8  `raf:"int8"`
			Int16 int16 `raf:"int16"`
			Int32 int32 `raf:"int32"`
			Int64 int64 `raf:"int64"`
		}

		testUnmarshalSame(t, MinInts{
			Int8:  math.MinInt8,
			Int16: math.MinInt16,
			Int32: math.MinInt32,
			Int64: math.MinInt64,
		})
	})
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
	err = Unmarshal(data, &b)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	expectedErr := "type mismatch for key num: expected int64, got string"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expectedErr)
	}
}

func TestUnmarshal_Failed_InvalidData(t *testing.T) {
	type A struct {
		Num int `raf:"num"`
	}

	data := []byte{0x01, 0x00, 0x01, 0x01, 0x00, 0x00}

	var a A
	err := Unmarshal(data, &a)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	expectedErr := "invalid RAF data"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expectedErr)
	}
}

func TestUnmarshalMap(t *testing.T) {
	primitives := getFullPrimitive()
	data, err := raf.Marshal(primitives)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := map[string]any{
		"str":        "Hello, RAF!",
		"int8":       int64(-8),
		"int16":      int64(-16),
		"int32":      int64(-32),
		"int64":      int64(-64),
		"int":        int64(-12345),
		"uint8":      int64(8),
		"uint16":     int64(16),
		"uint32":     int64(32),
		"uint64":     int64(64),
		"uint":       int64(12345),
		"float32":    float64(float32(3.14)),
		"float64":    float64(2.71828),
		"bool":       true,
		"null":       nil,
		"ptr_str":    "Pointer to string",
		"ptr_int":    int64(42),
		"ptr_null":   nil,
		"empty_str":  "",
		"empty_ptr":  "",
		"empty_int":  int64(0),
		"empty_bool": false,

		"ints":   []int64{-1, 0, 1},
		"ints8":  []int64{-8, 0, 8},
		"ints16": []int64{-16, 0, 16},
		"ints32": []int64{-32, 0, 32},
		"ints64": []int64{-64, 0, 64},

		"uints":   []int64{123, 123, 5, 66, 8, 9, 5},
		"uints8":  []int64{234, 16},
		"uints16": []int64{5, 5, 5, 5, 6},
		"uints32": []int64{9, 8, 7, 6, 5, 4, 3, 2, 1},
		"uints64": []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},

		"floats32": []float64{3.140000104904175, -2.7100000381469727},
		"floats64": []float64{2.71828, -3.14},
		"bools":    []bool{true, false, true},
		// "int_pointers": []*int{new(int(1)), new(int(2)), new(int(3)), nil},

		"strings": []string{"Hello", "RAF", "Test"},
	}

	var got map[string]any
	if err := Unmarshal(data, &got); err != nil {
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
