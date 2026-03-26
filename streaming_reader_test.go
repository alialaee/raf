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
}

func BenchmarkStreamBlock_Header(b *testing.B) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
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

func BenchmarkStreamBlock_Values(b *testing.B) {
	user := testUser{
		ID:      123,
		Name:    "Ali",
		Friends: []string{"Melina", "Pampam"},
		Active:  true,
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
			if _, err := sb.Next(); err != nil {
				b.Fatalf("Next error at index %d: %v", i, err)
			}
		}

		reader.Reset(data)
		sb.Reset(reader) // Reset for next iteration
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

	expectedKeys := [][]byte{[]byte("active"), []byte("friends"), []byte("id"), []byte("name")}
	expectedTypes := []Type{TypeBool, TypeArray, TypeInt64, TypeString}
	expectedValues := []any{user.Active, toBytesSlice(user.Friends), user.ID, []byte(user.Name)}

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

func toBytesSlice(strs []string) [][]byte {
	result := make([][]byte, len(strs))
	for i, s := range strs {
		result[i] = []byte(s)
	}
	return result
}
