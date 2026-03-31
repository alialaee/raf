package raf

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type testUser struct {
	ID      int
	Name    string
	Friends []string
	Active  bool
	Manager *testUser
}

func BenchmarkStreamBlock_Header(b *testing.B) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
		Manager: &testUser{
			ID:     456,
			Name:   "Bob",
			Active: false,
		},
	}
	data, err := Marshal(user)
	if err != nil {
		b.Fatalf("Marshal error: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))

	reader := bytes.NewReader(data)
	b.ResetTimer()
	for b.Loop() {
		if _, err := sb.ReadHeader(); err != nil {
			b.Fatalf("ReadHeader error: %v", err)
		}
		reader.Reset(data)
		sb.Reset(reader) // Reset for next iteration
	}
}

func BenchmarkBlock_Unmarshal(b *testing.B) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
		Manager: &testUser{
			ID:     456,
			Name:   "Bob",
			Active: false,
		},
	}
	data, err := Marshal(user)
	if err != nil {
		b.Fatalf("Marshal error: %v", err)
	}

	var result testUser
	b.ResetTimer()
	for b.Loop() {
		if err := Unmarshal(data, &result); err != nil {
			b.Fatalf("Unmarshal error: %v", err)
		}
	}
}

func BenchmarkStreamBlock_Values(b *testing.B) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
		Manager: &testUser{
			ID:     456,
			Name:   "Bob",
			Active: false,
		},
	}
	data, err := Marshal(user)
	if err != nil {
		b.Fatalf("Marshal error: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))

	reader := bytes.NewReader(data)
	b.ResetTimer()
	for b.Loop() {
		if _, err := sb.ReadHeader(); err != nil {
			b.Fatalf("ReadHeader error: %v", err)
		}

		for i := range sb.NumPairs() {
			switch sb.TypeAt(i) {
			case TypeMap:
				inner, err := sb.NextMap()
				if err != nil {
					b.Fatalf("NextMap error at index %d: %v", i, err)
				}
				// Read all values from the inner block to prevent parent block from hitting EOF.
				if _, err := inner.ReadHeader(); err != nil {
					b.Fatalf("inner ReadHeader error at index %d: %v", i, err)
				}
				for {
					if _, err := inner.Next(); errors.Is(err, io.EOF) {
						break
					} else if err != nil {
						b.Fatalf("inner Next error at index %d: %v", i, err)
					}
				}
			case TypeArray:
				inner, err := sb.NextArray()
				if err != nil {
					b.Fatalf("NextArray error at index %d: %v", i, err)
				}
				if err := inner.ReadHeader(); err != nil {
					b.Fatalf("inner ReadHeader error at index %d: %v", i, err)
				}
				for {
					if _, err := inner.Next(); errors.Is(err, io.EOF) {
						break
					} else if err != nil {
						b.Fatalf("inner Next error at index %d: %v", i, err)
					}
				}
			default:
				if _, err := sb.Next(); err != nil {
					b.Fatalf("Next error at index %d: %v", i, err)
				}
			}
		}

		reader.Reset(data)
		sb.Reset(reader)
	}
}

func TestStreamBlock_Header(t *testing.T) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
	}
	data, err := Marshal(user)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))

	expectedKeys := [][]byte{[]byte("active"), []byte("friends"), []byte("id"), []byte("manager"), []byte("name")}
	expectedTypes := []Type{TypeBool, TypeArray, TypeInt64, TypeMap, TypeString}
	expectedValues := []any{user.Active, toBytesSlice(user.Friends), user.ID, user.Manager, []byte(user.Name)}

	// Read header and verify keys
	keys, err := sb.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader error: %v", err)
	}
	if !testBytesEqual(t, keys, expectedKeys) {
		t.Errorf("Keys mismatch.\nGot: %v\nExpected: %v", keys, expectedKeys)
	}

	keys = sb.Keys()
	if !testBytesEqual(t, keys, expectedKeys) {
		t.Errorf("Keys mismatch on second call.\nGot: %v\nExpected: %v", keys, expectedKeys)
	}

	if sb.NumPairs() != len(expectedKeys) {
		t.Errorf("NumPairs mismatch.\nGot: %d\nExpected: %d", sb.NumPairs(), len(expectedKeys))
	}

	for i := range expectedTypes {
		if sb.TypeAt(i) != expectedTypes[i] {
			t.Errorf("Type mismatch at index %d.\nGot: %v\nExpected: %v", i, sb.TypeAt(i), expectedTypes[i])
		}
	}

	// Read values and verify
	for {
		val, err := sb.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		expectedVal := expectedValues[sb.nextValIndex-1]

		switch val.Type {
		case TypeBool:
			if val.Bool() != expectedVal.(bool) {
				t.Errorf("Value mismatch for bool.\nGot: %v\nExpected: %v", val.Data, expectedVal)
			}
		case TypeInt64:
			if val.Int64() != int64(expectedVal.(int)) {
				t.Errorf("Value mismatch for int64.\nGot: %v\nExpected: %v", val.Data, expectedVal)
			}
		case TypeString:
			if !bytes.Equal([]byte(val.String()), expectedVal.([]byte)) {
				t.Errorf("Value mismatch for string.\nGot: %s\nExpected: %s", val.Data, expectedVal)
			}
		case TypeMap:
		case TypeArray:
		}
	}
}

func testBytesEqual(t *testing.T, a, b [][]byte) bool {
	t.Helper()
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestStreamBlock_NextMap(t *testing.T) {
	type address struct {
		City    string
		Country string
	}
	type person struct {
		Address address
		Name    string
	}

	p := person{
		Address: address{City: "Amsterdam", Country: "NL"},
		Name:    "Ali",
	}
	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Sorted keys: "address" (TypeMap), "name" (TypeString)
	sb := NewStreamBlock(bytes.NewReader(data))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if sb.NumPairs() != 2 {
		t.Fatalf("expected 2 pairs, got %d", sb.NumPairs())
	}
	if sb.TypeAt(0) != TypeMap {
		t.Fatalf("expected TypeMap at index 0, got %s", sb.TypeAt(0))
	}

	inner, err := sb.NextMap()
	if err != nil {
		t.Fatalf("NextMap: %v", err)
	}

	// Inner block: sorted keys "city", "country"
	innerKeys, err := inner.ReadHeader()
	if err != nil {
		t.Fatalf("inner ReadHeader: %v", err)
	}
	expectedInnerKeys := [][]byte{[]byte("city"), []byte("country")}
	if !testBytesEqual(t, innerKeys, expectedInnerKeys) {
		t.Errorf("inner keys: got %q, want %q", innerKeys, expectedInnerKeys)
	}

	cityVal, err := inner.Next()
	if err != nil {
		t.Fatalf("inner Next (city): %v", err)
	}
	if cityVal.String() != p.Address.City {
		t.Errorf("city: got %q, want %q", cityVal.String(), p.Address.City)
	}

	// Leave "country" unread — parent should auto-drain on next Next().
	nameVal, err := sb.Next()
	if err != nil {
		t.Fatalf("parent Next (name): %v", err)
	}
	if nameVal.String() != p.Name {
		t.Errorf("name: got %q, want %q", nameVal.String(), p.Name)
	}
}

func TestStreamBlock_NextMap_TypeMismatch(t *testing.T) {
	user := testUser{ID: 1, Name: "Ali", Active: true}
	data, err := Marshal(user)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	// First key is "active" (TypeBool), not TypeMap.
	_, err = sb.NextMap()
	if err == nil {
		t.Fatal("expected error when calling NextMap on a non-map value")
	}
}

func TestStreamBlock_NextMap_SkipInner(t *testing.T) {
	type address struct {
		City string
	}
	type person struct {
		Address address
		Name    string
	}

	p := person{Address: address{City: "Berlin"}, Name: "Bob"}
	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	// Get the inner block but don't read anything from it.
	if _, err := sb.NextMap(); err != nil {
		t.Fatalf("NextMap: %v", err)
	}

	// Skip should auto-drain the inner block's bytes and then skip "name".
	if err := sb.Skip(); err != nil {
		t.Fatalf("Skip after unused NextMap: %v", err)
	}

	if _, err := sb.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after skipping all values, got %v", err)
	}
}

func toBytesSlice(strs []string) [][]byte {
	result := make([][]byte, len(strs))
	for i, s := range strs {
		result[i] = []byte(s)
	}
	return result
}

func TestStreamBlock_NextArray(t *testing.T) {
	type data struct {
		Names  []string
		Scores []int
		Active bool
	}

	d := data{
		Names:  []string{"Alice", "Bob"},
		Scores: []int{100, 200, 300},
		Active: true,
	}
	raw, err := Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Sorted keys: "active" (bool), "names" (array), "scores" (array)
	sb := NewStreamBlock(bytes.NewReader(raw))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if sb.NumPairs() != 3 {
		t.Fatalf("expected 3 pairs, got %d", sb.NumPairs())
	}

	// Skip "active"
	if err := sb.Skip(); err != nil {
		t.Fatalf("Skip active: %v", err)
	}

	// Read "names" array (dynamic-size elements)
	arr, err := sb.NextArray()
	if err != nil {
		t.Fatalf("NextArray (names): %v", err)
	}
	if err := arr.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader (names): %v", err)
	}
	if arr.ElemType() != TypeString {
		t.Fatalf("names elem type: got %s, want string", arr.ElemType())
	}
	if arr.Len() != len(d.Names) {
		t.Fatalf("names len: got %d, want %d", arr.Len(), len(d.Names))
	}
	for i, want := range d.Names {
		val, err := arr.Next()
		if err != nil {
			t.Fatalf("names Next[%d]: %v", i, err)
		}
		if got := val.String(); got != want {
			t.Errorf("names[%d]: got %q, want %q", i, got, want)
		}
	}
	if _, err := arr.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("names: expected EOF, got %v", err)
	}

	// Read "scores" array (fixed-size elements)
	arr2, err := sb.NextArray()
	if err != nil {
		t.Fatalf("NextArray (scores): %v", err)
	}
	if err := arr2.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader (scores): %v", err)
	}
	if arr2.Len() != len(d.Scores) {
		t.Fatalf("scores len: got %d, want %d", arr2.Len(), len(d.Scores))
	}
	for i, want := range d.Scores {
		val, err := arr2.Next()
		if err != nil {
			t.Fatalf("scores Next[%d]: %v", i, err)
		}
		if got := val.Int64(); got != int64(want) {
			t.Errorf("scores[%d]: got %d, want %d", i, got, want)
		}
	}

	// Should be EOF now
	if _, err := sb.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestStreamBlock_NextArray_TypeMismatch(t *testing.T) {
	user := testUser{ID: 1, Name: "Ali", Active: true}
	data, err := Marshal(user)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	// First key is "active" (TypeBool), not TypeArray.
	_, err = sb.NextArray()
	if err == nil {
		t.Fatal("expected error when calling NextArray on a non-array value")
	}
}

func TestStreamArray_NextMap(t *testing.T) {
	type pet struct {
		Name string
		Age  int
	}
	type owner struct {
		Pets []pet
	}

	o := owner{Pets: []pet{
		{Name: "Luna", Age: 3},
		{Name: "Max", Age: 5},
	}}
	data, err := Marshal(o)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(data))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	// "pets" is the only key, type array
	arr, err := sb.NextArray()
	if err != nil {
		t.Fatalf("NextArray: %v", err)
	}
	if err := arr.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader array: %v", err)
	}
	if arr.ElemType() != TypeMap {
		t.Fatalf("elem type: got %s, want map", arr.ElemType())
	}
	if arr.Len() != 2 {
		t.Fatalf("len: got %d, want 2", arr.Len())
	}

	// Read first pet via NextMap
	inner, err := arr.NextMap()
	if err != nil {
		t.Fatalf("NextMap[0]: %v", err)
	}
	if _, err := inner.ReadHeader(); err != nil {
		t.Fatalf("inner ReadHeader[0]: %v", err)
	}
	// Sorted keys: "age", "name"
	ageVal, err := inner.Next()
	if err != nil {
		t.Fatalf("inner Next age[0]: %v", err)
	}
	if ageVal.Int64() != int64(o.Pets[0].Age) {
		t.Errorf("pet[0] age: got %d, want %d", ageVal.Int64(), o.Pets[0].Age)
	}
	nameVal, err := inner.Next()
	if err != nil {
		t.Fatalf("inner Next name[0]: %v", err)
	}
	if nameVal.String() != o.Pets[0].Name {
		t.Errorf("pet[0] name: got %q, want %q", nameVal.String(), o.Pets[0].Name)
	}

	// Read second pet — partially read, let parent drain
	inner2, err := arr.NextMap()
	if err != nil {
		t.Fatalf("NextMap[1]: %v", err)
	}
	if _, err := inner2.ReadHeader(); err != nil {
		t.Fatalf("inner ReadHeader[1]: %v", err)
	}
	ageVal2, err := inner2.Next()
	if err != nil {
		t.Fatalf("inner Next age[1]: %v", err)
	}
	if ageVal2.Int64() != int64(o.Pets[1].Age) {
		t.Errorf("pet[1] age: got %d, want %d", ageVal2.Int64(), o.Pets[1].Age)
	}
	// Leave "name" unread — parent block should still be able to continue.

	// Parent should be at EOF
	if _, err := sb.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestStreamArray_NextMap_TypeMismatch(t *testing.T) {
	type data struct {
		Names []string
	}
	d := data{Names: []string{"a"}}
	raw, err := Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(raw))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	arr, err := sb.NextArray()
	if err != nil {
		t.Fatalf("NextArray: %v", err)
	}
	if err := arr.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader array: %v", err)
	}
	// Array of strings, not maps
	_, err = arr.NextMap()
	if err == nil {
		t.Fatal("expected error when calling NextMap on string array")
	}
}

func TestStreamArray_NextArray_TypeMismatch(t *testing.T) {
	type data struct {
		Names []string
	}
	d := data{Names: []string{"a"}}
	raw, err := Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	sb := NewStreamBlock(bytes.NewReader(raw))
	if _, err := sb.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	arr, err := sb.NextArray()
	if err != nil {
		t.Fatalf("NextArray: %v", err)
	}
	if err := arr.ReadHeader(); err != nil {
		t.Fatalf("ReadHeader array: %v", err)
	}
	// Array of strings, not arrays
	_, err = arr.NextArray()
	if err == nil {
		t.Fatal("expected error when calling NextArray on string array")
	}
}

func TestStreamBlock_NullBlock(t *testing.T) {
	// A null block is 6 zero bytes (version=0, size=0).
	nullBlock := make([]byte, 6)
	sb := NewStreamBlock(bytes.NewReader(nullBlock))

	keys, err := sb.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader null block: %v", err)
	}
	if !sb.IsNull() {
		t.Fatal("expected null block")
	}
	if len(keys) != 0 {
		t.Fatalf("null block: expected 0 keys, got %d", len(keys))
	}
	if sb.NumPairs() != 0 {
		t.Fatalf("null block: expected 0 pairs, got %d", sb.NumPairs())
	}

	// Cached — second call must return same result without re-reading.
	keys2, err := sb.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader null block (cached): %v", err)
	}
	if !sb.IsNull() || len(keys2) != 0 {
		t.Fatal("cached call changed null block result")
	}

	// Next on a null block should immediately return EOF.
	if _, err := sb.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next on null block: expected io.EOF, got %v", err)
	}
}
