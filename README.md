# raf

[![Test](https://github.com/alialaee/raf/actions/workflows/test.yml/badge.svg)](https://github.com/alialaee/raf/actions/workflows/test.yml)

`raf` provides a simple, read-optimized binary format in Go. 

It is designed for fast read access across a few kilobytes of data. Keys are sorted lexicographically as raw bytes to allow for quick retrieval. 

## Features

- **Read-optimized:** Designed for extremely fast sequential and random reads.
- **Compact:** Low overhead binary format.
- **Zero-dependency**

## Layout

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
