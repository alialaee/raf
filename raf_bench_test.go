package raf

import (
	"encoding/json"
	"fmt"
	"testing"
)

func BenchmarkBuilderBuild(b *testing.B) {

	keys := []KeyType{
		{"age", TypeInt64},
		{"city", TypeString},
		{"is_active", TypeBool},
		{"name", TypeString},
		{"score", TypeFloat64},
	}

	dst := make([]byte, 1024)
	builder := NewBuilder(dst)

	b.ReportAllocs()

	for b.Loop() {
		builder.Reset()

		builder.AddKeys(keys...)

		builder.AddInt64(30)
		builder.AddString("Berlin")
		builder.AddBool(true)
		builder.AddString("AliAlaee")
		builder.AddFloat64(99.5)

		var err error
		dst, err = builder.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(dst)), "serialized-bytes")
}

func BenchmarkBlockGet(b *testing.B) {
	builder := NewBuilder(nil)

	keys := make([]KeyType, 50)
	for i := range 50 {
		keys[i] = KeyType{
			Name: fmt.Sprintf("key%02d", i),
			Type: TypeInt64,
		}
	}

	builder.AddKeys(keys...)

	for i := range 50 {
		builder.AddInt64(int64(i))
	}

	dst, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	block := NewBlock(dst)
	searchKey := []byte("key25")

	b.ReportAllocs()

	b.ResetTimer()
	for b.Loop() {
		val, ok := block.Get(searchKey)
		if !ok || val.Type != TypeInt64 {
			b.Fatal("not found or wrong type")
		}
	}
}

func BenchmarkLookup(b *testing.B) {
	builder := NewBuilder(nil)

	keys := []KeyType{
		{"age", TypeInt64},
		{"city", TypeString},
		{"is_active", TypeBool},
		{"name", TypeString},
		{"score", TypeFloat64},
	}

	builder.AddKeys(keys...)

	builder.AddInt64(30)
	builder.AddString("Berlin")
	builder.AddBool(true)
	builder.AddString("AliAlaee")
	builder.AddFloat64(99.5)

	dst, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}
	block := NewBlock(dst)
	searchKey := []byte("score")

	b.ReportAllocs()

	for b.Loop() {
		val, ok := block.Get(searchKey)
		if !ok || val.Type != TypeFloat64 {
			b.Fatal("not found or wrong type")
		}
	}
}

func BenchmarkLookup_GoMap(b *testing.B) {
	m := map[string]any{
		"age":       int64(30),
		"city":      "Berlin",
		"is_active": true,
		"name":      "Ali Alaee",
		"score":     99.5,
	}

	b.ReportAllocs()

	for b.Loop() {
		val, ok := m["score"]
		if !ok {
			b.Fatal("not found")
		}
		if _, ok := val.(float64); !ok {
			b.Fatal("wrong type")
		}
	}
}

func BenchmarkExtract_JSON(b *testing.B) {
	payload := []byte(`{"age":30,"city":"Berlin","is_active":true,"name":"Ali Alaee","score":99.5}`)

	b.ReportMetric(float64(len(payload)), "serialized-bytes")
	b.ReportAllocs()

	for b.Loop() {
		var m map[string]any
		if err := json.Unmarshal(payload, &m); err != nil {
			b.Fatal(err)
		}

		val, ok := m["score"]
		if !ok {
			b.Fatal("not found")
		}
		if _, ok := val.(float64); !ok {
			b.Fatal("wrong type")
		}
	}
}

func BenchmarkArrayBuild(b *testing.B) {
	builder := NewBuilder(nil)

	ints := make([]int64, 100)
	for i := range ints {
		ints[i] = int64(i * 70000000)
	}

	strs := make([]string, 50)
	for i := range strs {
		strs[i] = "value_placeholder"
	}

	dst := make([]byte, 8192)

	b.ReportAllocs()

	for b.Loop() {
		builder.Reset()
		builder.AddKeys(
			KeyType{
				Name: "a_ints",
				Type: TypeArray,
			},
			KeyType{
				Name: "b_strs",
				Type: TypeArray,
			},
		)
		builder.AddInt64Array(ints...)
		builder.AddStringArray(strs...)

		var err error
		dst, err = builder.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(dst)), "serialized-bytes")
}

func BenchmarkArrayBuild_JSON(b *testing.B) {
	ints := make([]int64, 100)
	for i := range ints {
		ints[i] = int64(i * 70000000)
	}

	strs := make([][]byte, 50)
	for i := range strs {
		strs[i] = []byte("value_placeholder")
	}

	type data struct {
		Ints []int64
		Strs [][]byte
	}

	b.ReportAllocs()

	var out []byte
	for b.Loop() {
		data := data{
			Ints: ints,
			Strs: strs,
		}
		out, _ = json.Marshal(data)
	}
	b.ReportMetric(float64(len(out)), "serialized-bytes")
}

func BenchmarkArrayRead(b *testing.B) {
	builder := NewBuilder(nil)

	builder.AddKeys(
		KeyType{
			Name: "a_ints",
			Type: TypeArray,
		},
		KeyType{
			Name: "b_strs",
			Type: TypeArray,
		},
	)

	strs := []string{"string a", "string b", "string c", "string d"}
	ints := make([]int64, 100)
	for i := range ints {
		ints[i] = int64(i * 7)
	}

	builder.AddInt64Array(ints...)
	builder.AddStringArray(strs...)

	dst, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	block := NewBlock(dst)
	searchKey := []byte("a_ints")

	b.ReportAllocs()

	for b.Loop() {
		val, ok := block.Get(searchKey)
		if !ok || val.Type != TypeArray {
			b.Fatal("not found or wrong type")
		}
		arr := val.Array()
		// Access every element
		for i := range arr.Len() {
			_ = arr.At(i)
		}
	}
}

func BenchmarkMapBuild(b *testing.B) {
	innerDst := make([]byte, 512)
	outerDst := make([]byte, 1024)

	inner := NewBuilder(innerDst)
	outer := NewBuilder(outerDst)

	b.ReportAllocs()

	innerKeys := []KeyType{
		{"city", TypeString},
		{"name", TypeString},
	}

	outerKeys := []KeyType{
		{"age", TypeInt64},
		{"meta", TypeMap},
		{"score", TypeFloat64},
	}

	for b.Loop() {
		inner.Reset()
		inner.AddKeys(innerKeys...)

		inner.AddString("Berlin")
		inner.AddString("Ali")

		innerDst, err := inner.Build()
		if err != nil {
			b.Fatal(err)
		}

		outer.Reset()
		outer.AddKeys(outerKeys...)

		outer.AddInt64(30)
		outer.AddRaw(innerDst)
		outer.AddFloat64(99.5)

		outerDst, err = outer.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(outerDst)), "serialized-bytes")
}

func BenchmarkMapRead(b *testing.B) {
	innerDst := make([]byte, 512)
	outerDst := make([]byte, 1024)

	inner := NewBuilder(innerDst)
	outer := NewBuilder(outerDst)

	b.ReportAllocs()

	innerKeys := []KeyType{
		{"city", TypeString},
		{"name", TypeString},
	}

	outerKeys := []KeyType{
		{"age", TypeInt64},
		{"meta", TypeMap},
		{"score", TypeFloat64},
	}

	inner.AddKeys(innerKeys...)

	inner.AddString("Berlin")
	inner.AddString("Ali")

	innerDst, err := inner.Build()
	if err != nil {
		b.Fatal(err)
	}

	outer.AddKeys(outerKeys...)

	outer.AddInt64(30)
	outer.AddRaw(innerDst)
	outer.AddFloat64(99.5)

	outerDst, err = outer.Build()
	if err != nil {
		b.Fatal(err)
	}

	block := NewBlock(outerDst)
	metaKey := []byte("meta")
	nameKey := []byte("name")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		val, ok := block.Get(metaKey)
		if !ok || val.Type != TypeMap {
			b.Fatal("map not found")
		}
		innerBlock := val.Block()
		ival, iok := innerBlock.Get(nameKey)
		if !iok || ival.Type != TypeString || ival.String() != "Ali" {
			b.Fatal("inner value mismatch")
		}
	}
}
