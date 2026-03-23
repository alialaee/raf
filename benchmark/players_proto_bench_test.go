package main

import (
	"fmt"
	"testing"

	proto_playerpb "github.com/alialaee/raf/benchmark/proto"
	"google.golang.org/protobuf/proto"
)

var protoPlayers []proto_playerpb.Player = generateProtoPlayers(1000)

func BenchmarkProtobufMarshals_Player(b *testing.B) {
	b.Run("Protobuf", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := range b.N {
			_, err := proto.Marshal(&protoPlayers[i%len(protoPlayers)])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProtobufUnmarshals_Player(b *testing.B) {
	b.Run("Protobuf", func(b *testing.B) {
		marshaledPlayers := make([][]byte, len(protoPlayers))
		for i := range protoPlayers {
			data, err := proto.Marshal(&protoPlayers[i])
			if err != nil {
				b.Fatal(err)
			}
			marshaledPlayers[i] = data
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := range b.N {
			var player proto_playerpb.Player
			err := proto.Unmarshal(marshaledPlayers[i%len(protoPlayers)], &player)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func randomProtoPlayer() proto_playerpb.Player {
	level := rng.Intn(60) + 1
	items := make([]*proto_playerpb.PlayerItem, rng.Intn(10)+5)

	for i := range items {
		items[i] = randomProtoItem(proto_playerpb.ItemType(rng.Intn(int(proto_playerpb.ItemType_ITEM_TYPE_MISC) + 1)))
	}

	friends := make([]string, rng.Intn(5))
	for i := range friends {
		friends[i] = playerNames[rng.Intn(len(playerNames))]
	}

	return proto_playerpb.Player{
		Id:          int64(rng.Intn(1_000_000) + 1),
		Name:        fmt.Sprintf("%s-%d", playerNames[rng.Intn(len(playerNames))], rng.Intn(10_000)),
		Items:       items,
		Weapon:      randomProtoNamedArmor("Weapon"),
		Armors:      randomProtoArmors(),
		Health:      int64(rng.Intn(400) + 100),
		Mana:        int64(rng.Intn(250)),
		Level:       int64(level),
		CanFly:      rng.Intn(10) == 0,
		CanSwim:     rng.Intn(2) == 0,
		CanTeleport: rng.Intn(20) == 0,
		Friends:     friends,
	}
}

func randomProtoArmors() *proto_playerpb.PlayerArmors {
	return &proto_playerpb.PlayerArmors{
		Head:  randomProtoNamedArmor("Helm"),
		Body:  randomProtoNamedArmor("Chestplate"),
		Legs:  randomProtoNamedArmor("Greaves"),
		Arms:  randomProtoNamedArmor("Gauntlets"),
		Feet:  randomProtoNamedArmor("Boots"),
		Ring1: randomProtoNamedArmor("Ring"),
		Ring2: randomProtoNamedArmor("Ring"),
	}
}

func randomProtoNamedArmor(name string) *proto_playerpb.PlayerItem {
	return &proto_playerpb.PlayerItem{
		Id:   int64(rng.Intn(1_000_000) + 1),
		Name: name,
		Type: proto_playerpb.ItemType_ITEM_TYPE_ARMOR,
	}
}

var protoItemNamesByType = map[proto_playerpb.ItemType][]string{
	proto_playerpb.ItemType_ITEM_TYPE_WEAPON: {"Sword", "Axe", "Bow", "Dagger"},
	proto_playerpb.ItemType_ITEM_TYPE_ARMOR:  {"Helmet", "Chestplate", "Leggings", "Boots"},
	proto_playerpb.ItemType_ITEM_TYPE_POTION: {"Health Potion", "Mana Potion", "Stamina Potion"},
	proto_playerpb.ItemType_ITEM_TYPE_FOOD:   {"Bread", "Meat", "Fruit", "Vegetable"},
	proto_playerpb.ItemType_ITEM_TYPE_MISC:   {"Gemstone", "Scroll", "Key"},
}

func randomProtoItem(itemType proto_playerpb.ItemType) *proto_playerpb.PlayerItem {
	names := protoItemNamesByType[itemType]

	return &proto_playerpb.PlayerItem{
		Id:   int64(rng.Intn(1_000_000) + 1),
		Name: names[rng.Intn(len(names))],
		Type: itemType,
	}
}

func generateProtoPlayers(count int) []proto_playerpb.Player {
	records := make([]proto_playerpb.Player, count)
	for i := range records {
		records[i] = randomProtoPlayer()
	}
	return records
}
