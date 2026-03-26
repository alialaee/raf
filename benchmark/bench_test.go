package main

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/alialaee/raf"
	"github.com/fxamacker/cbor/v2"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var rngSource = rand.NewSource(42)
var rng = rand.New(rngSource)

type Marshaler func(v any) ([]byte, error)
type Unmarshaler func(data []byte, v any) error

func benchmarkMarshal[V any](b *testing.B, marshaler Marshaler, objects []V) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()
	for i := range b.N {
		_, err := marshaler(objects[i%len(objects)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkUnmarshal[V any](b *testing.B, marshaler Marshaler, unmarshaler Unmarshaler, objects []V) {
	b.Helper()
	marshaledObjects := make([][]byte, len(objects))
	for i, obj := range objects {
		data, err := marshaler(obj)
		if err != nil {
			b.Fatal(err)
		}
		marshaledObjects[i] = data
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		var obj V
		err := unmarshaler(marshaledObjects[i%len(objects)], &obj)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var rafMarshaledData []byte

func rafMarshalInto(v any) ([]byte, error) {
	var err error
	rafMarshaledData, err = raf.MarshalInto(v, rafMarshaledData[0:0])
	return rafMarshaledData, err
}

func benchmarkAllMarshals[V any](b *testing.B, objects []V) {
	b.Helper()
	marshalers := map[string]Marshaler{
		"RAF":             raf.Marshal,
		"RAF_MarshalInto": rafMarshalInto,
		"JSON":            json.Marshal,
		"MsgPack":         msgpack.Marshal,
		"CBOR":            cbor.Marshal,
		"BSON":            bson.Marshal,
	}

	for name, m := range marshalers {
		b.Run(name, func(b *testing.B) {
			benchmarkMarshal(b, m, objects)
		})
	}
}

func benchmarkAllUnmarshals[V any](b *testing.B, objects []V) {
	b.Helper()
	marshalers := map[string]Marshaler{
		"RAF":     raf.Marshal,
		"JSON":    json.Marshal,
		"MsgPack": msgpack.Marshal,
		"CBOR":    cbor.Marshal,
		"BSON":    bson.Marshal,
	}

	unmarshalers := map[string]Unmarshaler{
		"RAF":     raf.Unmarshal,
		"JSON":    json.Unmarshal,
		"MsgPack": msgpack.Unmarshal,
		"CBOR":    cbor.Unmarshal,
		"BSON":    bson.Unmarshal,
	}

	for name, m := range marshalers {
		u := unmarshalers[name]
		b.Run(name, func(b *testing.B) {
			benchmarkUnmarshal(b, m, u, objects)
		})
	}
}
