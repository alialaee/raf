package main

import "testing"

func BenchmarkBytes(b *testing.B) {
	buf := make([]byte, 0, 1024)
	for b.Loop() {
		copy(buf[:], "Hello, World!")
	}
}

func BenchmarkAddKeys(b *testing.B) {
	keyTypes := []KeyType{
		{Name: "name", Type: TypeString},
		{Name: "age", Type: TypeInt64},
		{Name: "score", Type: TypeFloat64},
		{Name: "active", Type: TypeBool},
		{Name: "tags", Type: TypeArray},
		{Name: "metadata", Type: TypeMap},
	}

	buf := make([]byte, 0, 1024)
	builder := NewBuilder(buf)

	b.ResetTimer()
	for b.Loop() {
		builder.Reset()
		builder.AddKeys(keyTypes...)
	}
}

func BenchmarkBuild_WithoutArrayAndMap(b *testing.B) {
	keyTypes := []KeyType{
		{Name: "name", Type: TypeString},
		{Name: "age", Type: TypeInt64},
		{Name: "score", Type: TypeFloat64},
		{Name: "active", Type: TypeBool},
		{Name: "tagsCount", Type: TypeInt64},
		{Name: "metadataCount", Type: TypeInt64},
		{Name: "friendName", Type: TypeString},
		{Name: "enemyName", Type: TypeString},
	}

	buf := make([]byte, 0, 1024)
	builder := NewBuilder(buf)

	b.ResetTimer()
	for b.Loop() {
		builder.Reset()

		builder.AddKeys(keyTypes...)

		builder.AddString("Ali")
		builder.AddInt64(int64(30))
		builder.AddFloat64(95.5)
		builder.AddBool(true)
		builder.AddInt64(3)
		builder.AddInt64(2)
		builder.AddString("Bob")
		builder.AddString("Eve")

		if _, err := builder.Build(); err != nil {
			b.Fatal(err)
		}
	}

}
