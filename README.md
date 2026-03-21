# raf

[![Test](https://github.com/alialaee/raf/actions/workflows/test.yml/badge.svg)](https://github.com/alialaee/raf/actions/workflows/test.yml)

`raf` provides a simple, read-optimized binary format in Go.

Designed for fast read access across a few kilobytes of data, keys are sorted lexicographically as raw bytes for quick retrieval. RAF can represent what JSON can.

> [!NOTE]  
> Originally part of a proprietary database engine, this was extracted into a standalone library to improve its ergonomics and add new features. Also take a look at its sister project for writing and reading sequential data (logs), [logfile](https://github.com/alialaee/logfile).


## Features

- **Type-rich:** Supports types similar to JSON, see [Differences from JSON](#differences-from-json).
- **Read-optimized:** Built for extremely fast sequential and random reads.
- **Random lookup:** Retrieve specific fields without full deserialization.
- **Simple:** The format is straightforward to parse and implement.
- **Compact:** Low-overhead binary format.
- **Schema-less**
- **Canonical serialization:** Only one representation of a single data.
- **Zero-allocation** by using `Builder` and `Block` and minimal allocations using `Marshal` and `Unmarshal`.

## Differences from JSON

While `raf` is type-rich and flexible like JSON, it has two key structural differences by design:

1. **Root must be a map:** Unlike JSON, where the root can be any value, a valid `raf` payload must always have a map (key-value pairs) at its root.
2. **Homogeneous arrays:** Arrays in `raf` must contain elements of the exact same type. You cannot mix types (e.g., strings and integers) within a single array.

## Format specification

For details on the exact binary layout, see [`raf.go`](raf.go).

## Benchmarking

See the [benchmark](benchmark) directory for performance comparisons against a few reflection-based encoders/decoders.

Here's a summary of the results on my machine (Apple MacBook Air M4):

```
goos: darwin
goarch: arm64
pkg: github.com/alialaee/raf/benchmark
cpu: Apple M4

BenchmarkRAF_Marshal-10         1277 ns/op	    2016 B/op	       2 allocs/op
BenchmarkMsgPack_Marshal-10     1860 ns/op	    2327 B/op	       6 allocs/op
BenchmarkJSON_Marshal-10        1297 ns/op	    1355 B/op	       2 allocs/op
BenchmarkCBOR_Marshal-10        1126 ns/op	    1046 B/op	       2 allocs/op
BenchmarkBSON_Marshal-10        2763 ns/op	    1391 B/op	       2 allocs/op

BenchmarkRAF_Unmarshal-10       947.8 ns/op	     913 B/op	      25 allocs/op
BenchmarkRAF_Lookup_Name-10     17.63 ns/op	       0 B/op	       0 allocs/op
BenchmarkJSON_Unmarshal-10      7816 ns/op	    1744 B/op	      35 allocs/op
BenchmarkMsgPack_Unmarshal-10   3223 ns/op	    1290 B/op	      28 allocs/op
BenchmarkCBOR_Unmarshal-10      3419 ns/op	     914 B/op	      25 allocs/op
BenchmarkBSON_Unmarshal-10      6073 ns/op	    2839 B/op	     154 allocs/op
```

## Example Usage

You can use `raf` in two ways:

- Using `Marshal` and `Unmarshal` for general cases.
- Using `Builder` and `Block` for higher performance use-cases. Using `Block` let's you to lookup specific fields without deserializing the whole payload.

### Marshaling and Unmarshaling

`raf` supports encoding and decoding Go structs and maps using `Marshal` and `Unmarshal`, similar to `encoding/json`.

```go
package main

import (
	"fmt"
	"github.com/alialaee/raf"
)

type User struct {
	ID       int64    `raf:"id"`
	Name     string   `raf:"name"`
	IsActive bool     `raf:"is_active"`
	Roles    []string `raf:"roles"`
}

func main() {
	user := User{
		ID:       1,
		Name:     "Ali",
		IsActive: true,
		Roles:    []string{"admin", "user"},
	}

	// Encode to raf binary format
	data, err := raf.Marshal(user)
	if err != nil {
		panic(err)
	}

	// Decode back to a struct
	var decoded User
	if err := raf.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}

	fmt.Printf("Decoded: %+v\n", decoded)
}
```

### Builder and Block

Use `raf.Builder` to construct your payload and `raf.Block` to read it. `Block` let's you to lookup specific fields without deserializing the whole payload.

```go
import (
	"fmt"
	"github.com/alialaee/raf"
)
func Build() []byte {
	b := raf.NewBuilder(nil)
	b.AddKeys(
		raf.KeyType{
			Name: "a_string",
			Type: raf.TypeString,
		},
		raf.KeyType{
			Name: "b_int64",
			Type: raf.TypeInt64,
		},
		raf.KeyType{
			Name: "c_bool",
			Type: raf.TypeBool,
		},
		raf.KeyType{
			Name: "d_map",
			Type: raf.TypeMap,
		},
		raf.KeyType{
			Name: "e_array",
			Type: raf.TypeArray,
		},
	)

	b.AddString("raf")
	b.AddInt64(1)
	b.AddBool(true)

	// Let's add a map
	err := b.AddBuilderFn(func(b *raf.Builder) error {
		b.AddKeys(
			raf.KeyType{
				Name: "author",
				Type: raf.TypeString,
			},
		)
		b.AddString("ali")
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Let's add an array
	err = b.AddArrayFn(raf.TypeString, 3, func(b *raf.ArrayBuilder) error {
		b.AddString("admin")
		b.AddString("user")
		b.AddString("guest")
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Build into a byte slice
	buf, err := b.Build()
	if err != nil {
		panic(err)
	}

	return buf
}

func Read(buf []byte) {
	block := raf.NewBlock(buf)
	if !block.Valid() {
		panic("invalid payload")
	}

	// Look up by key directly
	val, ok := block.Get([]byte("a_string"))
	if ok && val.Type == raf.TypeString {
		fmt.Printf("a_string: %s\n", val.String())
	}

	val, ok = block.Get([]byte("b_int64"))
	if ok && val.Type == raf.TypeInt64 {
		fmt.Printf("b_int64: %d\n", val.Int64())
	}

	val, ok = block.Get([]byte("c_bool"))
	if ok && val.Type == raf.TypeBool {
		fmt.Printf("c_bool: %t\n", val.Bool())
	}

	val, ok = block.Get([]byte("d_map"))
	if ok && val.Type == raf.TypeMap {
		d_map := val.Block()
		val, ok = d_map.Get([]byte("author"))
		if ok && val.Type == raf.TypeString {
			fmt.Printf("DMap:\n\tAuthor: %s\n", val.String())
		}
	}

	val, ok = block.Get([]byte("e_array"))
	if ok && val.Type == raf.TypeArray {
		e_array := val.Array()
		fmt.Printf("EArray:\n")
		for i := 0; i < e_array.Len(); i++ {
			fmt.Printf("\t%d: %s\n", i, e_array.AtString(i, nil))
		}
	}

}
```

## Installation

```sh
go get github.com/alialaee/raf
```
