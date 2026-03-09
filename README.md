# raf

[![Test](https://github.com/alialaee/raf/actions/workflows/test.yml/badge.svg)](https://github.com/alialaee/raf/actions/workflows/test.yml)

`raf` provides a simple, read-optimized binary format in Go.

Designed for fast read access across a few kilobytes of data, keys are sorted lexicographically as raw bytes for quick retrieval.

> [!NOTE]  
> Originally part of a proprietary database engine, this was extracted into a standalone library to improve its ergonomics and add new features.

## Features

- **Read-optimized:** Built for extremely fast sequential and random reads.
- **Random lookup:** Retrieve specific fields without full deserialization.
- **Simple:** The format is straightforward to parse and implement.
- **Compact:** Low-overhead binary format.
- **Type-rich:** Supports types similar to JSON, see [Differences from JSON](#differences-from-json).
- **Schema-less**
- **Zero-dependency**
- **Canonical serialization**
- **Zero-allocation**

## Goals

- Prioritize read performance and random access.
- Keep the format simple to understand.
- Canonical serialization (only one representation of a single data).
- Minimal allocations.
- Be suitable for use both on the wire and on disk.
- Provide a highly ergonomic API.
- Keep schemas optional.

## Non-Goals

- Streaming and support for large datasets.
- Unions and other complex, high-level data types.

## Differences from JSON

While `raf` is type-rich and flexible like JSON, it has two key structural differences by design:

1. **Root must be a map:** Unlike JSON, where the root can be any value, a valid `raf` payload must always have a map (key-value pairs) at its root.
2. **Homogeneous arrays:** Arrays in `raf` must contain elements of the exact same type. You cannot mix types (e.g., strings and integers) within a single array.

## Format specification

For details on the exact binary layout, see [`raf.go`](raf.go).

## Example Usage

### Building a Payload

Use `raf.Builder` to construct your payload. It allocates memory and handles offsets.

```go
package main

import (
	"fmt"
	"github.com/alialaee/raf"
)

func main() {
	b := raf.NewBuilder()

	// Keys are automatically sorted during build
	b.AddString([]byte("name"), []byte("raf"))
	b.AddInt64([]byte("version"), 1)
	b.AddBool([]byte("fast"), true)

	// You can add nested fields as well
	nested := raf.NewBuilder()
	nested.AddString([]byte("author"), []byte("ali"))
	nestedBuf, _ := nested.Build(nil)
	b.AddMap([]byte("metadata"), nestedBuf)

	// Build into a byte slice
	buf, err := b.Build(nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payload size: %d bytes\n", len(buf))
}
```

### Reading a Payload

Given a byte slice, you can quickly look up specific fields by casting it to `raf.Block` and using the `Get` method, without deserializing everything.

```go
	block := raf.Block(buf)
	if !block.Valid() {
		panic("invalid payload")
	}

	// Look up by key directly
	val, ok := block.Get([]byte("name"))
	if ok && val.Type == raf.TypeString {
		fmt.Printf("Name: %s\n", val.String())
	}
```

## Installation

```sh
go get github.com/alialaee/raf
```
