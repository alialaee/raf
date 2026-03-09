package raf

import (
	"encoding/json"
	"testing"
)

func BenchmarkBuilderBuild(b *testing.B) {
	builder := NewBuilder()

	keys := [][]byte{
		[]byte("age"),
		[]byte("city"),
		[]byte("is_active"),
		[]byte("name"),
		[]byte("score"),
	}

	dst := make([]byte, 1024)

	b.ReportAllocs()

	for b.Loop() {
		builder.Reset()
		builder.AddInt64(keys[0], 30)
		builder.AddString(keys[1], []byte("Berlin"))
		builder.AddBool(keys[2], true)
		builder.AddString(keys[3], []byte("Ali Alaee"))
		builder.AddFloat64(keys[4], 99.5)

		var err error
		dst, err = builder.Build(dst)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(dst)), "serialized-bytes")
}

func BenchmarkBlockGet(b *testing.B) {
	builder := NewBuilder()

	for i := range byte(50) {
		builder.AddInt64([]byte{i, 'k', 'e', 'y'}, int64(i))
	}

	dst, err := builder.Build(nil)
	if err != nil {
		b.Fatal(err)
	}

	block := Block(dst)
	searchKey := []byte{25, 'k', 'e', 'y'}

	b.ReportAllocs()

	for b.Loop() {
		vt, _, ok := block.Get(searchKey)
		if !ok || vt != TypeInt64 {
			b.Fatal("not found or wrong type")
		}
	}
}

func BenchmarkLookup_FlatKV(b *testing.B) {
	builder := NewBuilder()
	builder.AddInt64([]byte("age"), 30)
	builder.AddString([]byte("city"), []byte("Berlin"))
	builder.AddBool([]byte("is_active"), true)
	builder.AddString([]byte("name"), []byte("Ali Alaee"))
	builder.AddFloat64([]byte("score"), 99.5)

	dst, err := builder.Build(nil)
	if err != nil {
		b.Fatal(err)
	}
	block := Block(dst)
	searchKey := []byte("score")

	b.ReportAllocs()

	for b.Loop() {
		vt, _, ok := block.Get(searchKey)
		if !ok || vt != TypeFloat64 {
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
	builder := NewBuilder()

	ints := make([]int64, 100)
	for i := range ints {
		ints[i] = int64(i * 70000000)
	}

	strs := make([][]byte, 50)
	for i := range strs {
		strs[i] = []byte("value_placeholder")
	}

	dst := make([]byte, 8192)

	b.ReportAllocs()

	for b.Loop() {
		builder.Reset()
		builder.AddInt64Array([]byte("a_ints"), ints)
		builder.AddStringArray([]byte("b_strs"), strs)

		var err error
		dst, err = builder.Build(dst)
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
	builder := NewBuilder()

	ints := make([]int64, 100)
	for i := range ints {
		ints[i] = int64(i * 7)
	}

	builder.AddInt64Array([]byte("a_ints"), ints)
	builder.AddStringArray([]byte("b_strs"), [][]byte{
		[]byte("hello"), []byte("world"), []byte("foo"), []byte("bar"),
	})

	dst, err := builder.Build(nil)
	if err != nil {
		b.Fatal(err)
	}

	block := Block(dst)
	searchKey := []byte("a_ints")

	b.ReportAllocs()

	for b.Loop() {
		vt, vb, ok := block.Get(searchKey)
		if !ok || vt != TypeArray {
			b.Fatal("not found or wrong type")
		}
		arr := Array(vb)
		// Access every element
		for i := range arr.Len() {
			_ = arr.At(i)
		}
	}
}

func BenchmarkMapBuild(b *testing.B) {
	inner := NewBuilder()
	outer := NewBuilder()
	innerDst := make([]byte, 512)
	outerDst := make([]byte, 1024)

	b.ReportAllocs()

	for b.Loop() {
		inner.Reset()
		inner.AddString([]byte("city"), []byte("Berlin"))
		inner.AddString([]byte("name"), []byte("Ali"))

		var err error
		innerDst, err = inner.Build(innerDst)
		if err != nil {
			b.Fatal(err)
		}

		outer.Reset()
		outer.AddInt64([]byte("age"), 30)
		outer.AddMap([]byte("meta"), innerDst)
		outer.AddFloat64([]byte("score"), 99.5)

		outerDst, err = outer.Build(outerDst)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(outerDst)), "serialized-bytes")
}

func BenchmarkMapRead(b *testing.B) {
	inner := NewBuilder()
	inner.AddString([]byte("city"), []byte("Berlin"))
	inner.AddString([]byte("name"), []byte("Ali"))
	innerDst, _ := inner.Build(nil)

	outer := NewBuilder()
	outer.AddInt64([]byte("age"), 30)
	outer.AddMap([]byte("meta"), innerDst)
	outer.AddFloat64([]byte("score"), 99.5)
	outerDst, _ := outer.Build(nil)

	block := Block(outerDst)
	metaKey := []byte("meta")
	nameKey := []byte("name")

	b.ReportAllocs()

	for b.Loop() {
		vt, vb, ok := block.Get(metaKey)
		if !ok || vt != TypeMap {
			b.Fatal("map not found")
		}
		innerBlock := Block(vb)
		ivt, ivb, iok := innerBlock.Get(nameKey)
		if !iok || ivt != TypeString || string(ivb) != "Ali" {
			b.Fatal("inner value mismatch")
		}
	}
}
