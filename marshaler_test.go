package raf

import (
	"reflect"
	"testing"
)

type User struct {
	ID       int64    `raf:"id"`
	Name     string   `raf:"name"`
	IsActive bool     `raf:"is_active"`
	Score    float64  `raf:"score"`
	Ignored  string   `raf:"-"`
	Roles    []string `raf:"roles"`
}

func TestMarshalUnmarshalStruct(t *testing.T) {
	orig := User{
		ID:       123,
		Name:     "Ali",
		IsActive: true,
		Score:    99.5,
		Ignored:  "ignore me",
		Roles:    []string{"admin", "user"},
	}

	data, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var dst User
	err = Unmarshal(data, &dst)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	orig.Ignored = ""
	if !reflect.DeepEqual(orig, dst) {
		t.Errorf("Mismatch.\nwant: %+v\ngot:  %+v", orig, dst)
	}
}

func TestMarshalUnmarshalMap(t *testing.T) {
	orig := map[string]any{
		"id":        int64(456),
		"name":      "Bob",
		"is_active": false,
		"score":     88.5,
	}

	data, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal map failed: %v", err)
	}

	dst := make(map[string]any)
	err = Unmarshal(data, &dst)
	if err != nil {
		t.Fatalf("Unmarshal map failed: %v", err)
	}

	if !reflect.DeepEqual(orig, dst) {
		t.Errorf("Mismatch.\nwant: %+v\ngot:  %+v", orig, dst)
	}
}

func TestMarshalUnmarshalPointer(t *testing.T) {
	orig := &User{
		ID:       999,
		Name:     "Pointer",
		IsActive: true,
	}

	data, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal pointer failed: %v", err)
	}

	var dst *User
	err = Unmarshal(data, &dst)
	if err != nil {
		t.Fatalf("Unmarshal pointer failed: %v", err)
	}

	if dst == nil {
		t.Fatal("Expected dst to be non-nil")
	}

	if !reflect.DeepEqual(*orig, *dst) {
		t.Errorf("Mismatch.\nwant: %+v\ngot:  %+v", *orig, *dst)
	}
}

func TestMarshalUnmarshalEmptyArray(t *testing.T) {
	orig := User{
		Roles: []string{}, // Empty array
	}

	data, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal empty array failed: %v", err)
	}

	var dst User
	err = Unmarshal(data, &dst)
	if err != nil {
		t.Fatalf("Unmarshal empty array failed: %v", err)
	}

	// Because zero length slices can become nil, DeepEqual might fail if we compare directly nil vs []string{}.
	// Just check length.
	if len(dst.Roles) != 0 {
		t.Errorf("Expected 0 roles, got %d", len(dst.Roles))
	}
}

type Inner struct {
	Value int64 `raf:"value"`
}

type AllStruct struct {
	Name     string            `raf:"name"`
	Raw      []byte            `raf:"raw"`
	Nested   Inner             `raf:"nested"`
	NilPtr   *string           `raf:"nil_ptr"`
	Scores   []float64         `raf:"scores"`
	Flags    []bool            `raf:"flags"`
	Meta     map[string]string `raf:"meta"`
	Untagged string
}

func TestMarshalUnmarshalAllStruct(t *testing.T) {
	str := "ptr value"
	orig := AllStruct{
		Name:     "Complex",
		Raw:      []byte{1, 2, 3, 4},
		Nested:   Inner{Value: 42},
		NilPtr:   nil, // Test nil pointer
		Scores:   []float64{1.1, 2.2, 3.3},
		Flags:    []bool{true, false, true},
		Meta:     map[string]string{"key1": "val1", "key2": "val2"},
		Untagged: "untagged_val",
	}

	data, err := Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var dst AllStruct
	err = Unmarshal(data, &dst)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(orig, dst) {
		t.Errorf("Mismatch.\nwant: %#v\ngot:  %#v", orig, dst)
	}

	// Also test with non-nil pointer
	orig.NilPtr = &str
	data, err = Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal with ptr failed: %v", err)
	}

	var dstPtr AllStruct
	err = Unmarshal(data, &dstPtr)
	if err != nil {
		t.Fatalf("Unmarshal with ptr failed: %v", err)
	}

	// We must compare string pointer contents or use deep equal carefully
	if dstPtr.NilPtr == nil || *dstPtr.NilPtr != *orig.NilPtr {
		t.Errorf("Mismatch with ptr.\nwant: %v\ngot:  %v", *orig.NilPtr, dstPtr.NilPtr)
	}

	// Set ptrs to nil to pass deep equal
	dstPtr.NilPtr = nil
	orig.NilPtr = nil
	if !reflect.DeepEqual(orig, dstPtr) {
		t.Errorf("Mismatch with ptr base.\nwant: %#v\ngot:  %#v", orig, dstPtr)
	}
}

func TestMarshalErrors(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"unsupported_int", 42},
		{"non_string_key_map", map[int]string{1: "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Marshal(tt.input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}
