package r2

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/alialaee/raf"
)

var opsCache sync.Map // map[reflect.Type][]opSet

type Contact struct {
	Email string `raf:"email"`
	Phone string `raf:"phone"`
}

type Player struct {
	ID     int     `raf:"id"`
	Name   string  `raf:"name"`
	Age    *int    `raf:"age"`
	Alive  bool    `raf:"alive"`
	Happy  bool    `raf:"happy"`
	Weight float64 `raf:"weight"`
	Mother string  `raf:"mother"`

	Contacts []Contact `raf:"contact"`

	Friends   []string `raf:"friends"`
	FrindsAge []int    `raf:"friends_age"`
}

var pToMarshal = Player{
	ID:     12345,
	Name:   "Alice",
	Age:    new(30),
	Alive:  true,
	Happy:  false,
	Weight: 150.5,
	Mother: "Eve",
	Contacts: []Contact{
		{
			Email: "alice@alice.com",
			Phone: "123-456-7890",
		},
		{
			Email: "bob@bob.com",
			Phone: "987-654-3210",
		},
	},
	Friends:   []string{"Bob", "Charlie"},
	FrindsAge: []int{25, 28},
}

func BenchmarkOpCodeUnmarshal(b *testing.B) {
	data, err := raf.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	dec := NewUnmarshaler()

	p := Player{}
	for b.Loop() {
		if err := dec.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	if !reflect.DeepEqual(p, pToMarshal) {
		b.Fatalf("unexpected result: got %+v, want %+v", p, pToMarshal)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	data, err := raf.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	var p Player
	b.ResetTimer()
	for b.Loop() {
		if err := raf.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	data, err := json.Marshal(pToMarshal)
	if err != nil {
		b.Fatal(err)
	}

	var p Player
	b.ResetTimer()
	for b.Loop() {
		if err := json.Unmarshal(data, &p); err != nil {
			b.Fatal(err)
		}
	}
}
