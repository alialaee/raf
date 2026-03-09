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
- **Type-rich:** Supports types similar to JSON, see [Differences from JSON](#json-diff).
- **Schema-less**
- **Zero-dependency**
- **Canonical serialization**

## Goals

- Prioritize read performance and random access.
- Keep the format simple to understand.
- Canonical serialization (only one representation of a single data).
- Be suitable for use both on the wire and on disk.
- Provide a highly ergonomic API.
- Keep schemas optional.

## Non-Goals

- Streaming and support for large datasets.
- Unions and other complex, high-level data types.

## Differences from JSON {#json-diff}

While `raf` is type-rich and flexible like JSON, it has two key structural differences by design:

1. **Root must be a map:** Unlike JSON, where the root can be any value, a valid `raf` payload must always have a map (key-value pairs) at its root.
2. **Homogeneous arrays:** Arrays in `raf` must contain elements of the exact same type. You cannot mix types (e.g., strings and integers) within a single array.

## Format specification

The binary format uses a continuous buffer with the following structure:

- Header (Version, Size, Count)
- Key offsets (relative)
- Value types (1 byte per value)
- Value offsets (relative, for variable-length items)
- Keys buffer
- Values buffer

For more details on the exact binary layout, see [`raf.go`](raf.go).

## Installation

```sh
go get github.com/alialaee/raf
```
