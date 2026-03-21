package main

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

func main() {
	buf := Build()
	fmt.Printf("Payload size: %d bytes\n", len(buf))
	Read(buf)
}
