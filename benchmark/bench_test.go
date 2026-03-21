package main

import (
	"encoding/json"
	"testing"

	"github.com/alialaee/raf"
	"github.com/fxamacker/cbor/v2"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var players []Player

func init() {
	players = createRecords(1000)
}

func BenchmarkRAF_Marshal(b *testing.B) {
	for i := range b.N {
		_, err := raf.Marshal(players[i%len(players)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMsgPack_Marshal(b *testing.B) {
	for i := range b.N {
		_, err := msgpack.Marshal(players[i%len(players)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSON_Marshal(b *testing.B) {
	for i := range b.N {
		_, err := json.Marshal(players[i%len(players)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBOR_Marshal(b *testing.B) {
	for i := range b.N {
		_, err := cbor.Marshal(players[i%len(players)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBSON_Marshal(b *testing.B) {
	for i := range b.N {
		_, err := bson.Marshal(players[i%len(players)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRAF_Unmarshal(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := raf.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		var p Player
		err := raf.Unmarshal(marshaledPlayers[i%len(players)], &p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRAF_Lookup_Name(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := raf.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		block := raf.NewBlock(marshaledPlayers[i%len(players)])
		_, found := block.Get([]byte("name"))
		if !found {
			b.Fatal("key not found")
		}
	}

}

func BenchmarkJSON_Unmarshal(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := json.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		var p Player
		err := json.Unmarshal(marshaledPlayers[i%len(players)], &p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMsgPack_Unmarshal(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := msgpack.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		var p Player
		err := msgpack.Unmarshal(marshaledPlayers[i%len(players)], &p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBOR_Unmarshal(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := cbor.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		var p Player
		err := cbor.Unmarshal(marshaledPlayers[i%len(players)], &p)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBSON_Unmarshal(b *testing.B) {
	marshaledPlayers := make([][]byte, len(players))
	for i, p := range players {
		data, err := bson.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		marshaledPlayers[i] = data
	}

	b.ResetTimer()
	for i := range b.N {
		var p Player
		err := bson.Unmarshal(marshaledPlayers[i%len(players)], &p)
		if err != nil {
			b.Fatal(err)
		}
	}
}
