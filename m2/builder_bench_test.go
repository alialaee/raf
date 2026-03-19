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

func BenchmarkArrayBuilder_Int64(b *testing.B) {
	buf := make([]byte, 0, 256)
	ab := NewArrayBuilder(buf, TypeInt64, 5)

	b.ResetTimer()
	for b.Loop() {
		ab.Reset(TypeInt64, 5)
		ab.AddInt64(10)
		ab.AddInt64(20)
		ab.AddInt64(30)
		ab.AddInt64(40)
		ab.AddInt64(50)
		if _, err := ab.Build(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArrayBuilder_String(b *testing.B) {
	buf := make([]byte, 0, 256)
	ab := NewArrayBuilder(buf, TypeString, 4)

	b.ResetTimer()
	for b.Loop() {
		ab.Reset(TypeString, 4)
		ab.AddString("hello")
		ab.AddString("world")
		ab.AddString("foo")
		ab.AddString("bar")
		if _, err := ab.Build(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuild_WithArrayAndMap(b *testing.B) {
	keyTypes := []KeyType{
		{Name: "name", Type: TypeString},
		{Name: "age", Type: TypeInt64},
		{Name: "tags", Type: TypeArray},
		{Name: "metadata", Type: TypeMap},
	}

	buf := make([]byte, 0, 1024)
	builder := NewBuilder(buf)

	b.ResetTimer()
	for b.Loop() {
		builder.Reset()
		builder.AddKeys(keyTypes...)

		builder.AddString("Ali")
		builder.AddInt64(30)

		err := builder.AddArrayFn(TypeString, 3, func(ab *ArrayBuilder) {
			ab.AddString("go")
			ab.AddString("rust")
			ab.AddString("zig")
		})
		if err != nil {
			b.Fatal(err)
		}

		err = builder.AddBuilderFn(func(mb *Builder) error {
			mb.AddKeys([]KeyType{
				{Name: "score", Type: TypeFloat64},
				{Name: "active", Type: TypeBool},
			}...)
			mb.AddFloat64(95.5)
			mb.AddBool(true)
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}

		if _, err := builder.Build(); err != nil {
			b.Fatal(err)
		}
	}
}
