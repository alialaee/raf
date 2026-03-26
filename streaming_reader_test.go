package raf

import (
	"bytes"
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
