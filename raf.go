// Package flatkv provides a simple, read-optimized binary format for key-value pairs.
// Keys are sorted lexicographically as raw bytes to allow fast retrieval.
// It's designed for a few kilobytes of data, with a focus on fast read access.
// All integer fields are stored in big-endian format.
//
// Layout:
//
//	[u8]  Version (e.g., 0x01)
//	[u16] Total data size
//	[u8]  Number of pairs (N)
//	[u16 * (N+1)] Array of key offsets (relative to start of key bytes)
//	[u8  * N]     Array of value types
//	[u16 * (N+1)] Array of value offsets (relative to start of value bytes)
//	[...u8]       Array of key bytes
//	[...u8]       Array of value bytes
//
// Value Types (1 byte):
//
//	0x01: string      0x04: bool
//	0x02: int64       0x05: array
//	0x03: float64     0x06: map (value is the same as the Layout, it's recursive)
//	0xff: null
//
// Arrays:
//
//	[u8] Type of array elements (same as value types above)
//	[u16] Entries in the array
//	[u16 * (N+1)] Offsets for the values only if the type is dynamically sized (e.g., string).
//
// Notes:
// - Keys: Must be unique, raw byte arrays, and ordered by byte value (not locale-aware).
// - Strings: Raw byte arrays. Decoding (e.g., UTF-8) is left to the client. Zero-length strings are permitted.
// - Booleans: 1 byte long. 0x00 is false, any other value is true.
// - Nulls: The null value type (0xff) has a zero-length entry (identical adjacent offsets) in the value offsets array.
package raf
