package main

import (
	"fmt"
	"testing"

	"github.com/alialaee/raf"
)

var players []Player = generatePlayers(1000)

func BenchmarkAllMarshals_Player(b *testing.B) {
	benchmarkAllMarshals(b, players)
}

func BenchmarkAllUnmarshals_Player(b *testing.B) {
	benchmarkAllUnmarshals(b, players)
}

func BenchmarkRAF_Lookup_Players(b *testing.B) {
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

func BenchmarkRAF_Marshal_Manual_Player(b *testing.B) {
	b.ResetTimer()

	builder := raf.NewBuilder(nil)
	for i := range b.N {
		builder.Reset()
		player := players[i%len(players)]

		builder.AddKeys(
			raf.KeyType{Name: "Armors", Type: raf.TypeMap},
			raf.KeyType{Name: "CanFly", Type: raf.TypeBool},
			raf.KeyType{Name: "CanSwim", Type: raf.TypeBool},
			raf.KeyType{Name: "CanTeleport", Type: raf.TypeBool},
			raf.KeyType{Name: "Friends", Type: raf.TypeArray},
			raf.KeyType{Name: "Health", Type: raf.TypeInt64},
			raf.KeyType{Name: "ID", Type: raf.TypeInt64},
			raf.KeyType{Name: "Items", Type: raf.TypeArray},
			raf.KeyType{Name: "Level", Type: raf.TypeInt64},
			raf.KeyType{Name: "Mana", Type: raf.TypeInt64},
			raf.KeyType{Name: "Name", Type: raf.TypeString},
			raf.KeyType{Name: "Weapon", Type: raf.TypeMap},
		)

		// Armors
		builder.AddBuilderFn(func(b *raf.Builder) error {
			b.AddKeys(
				raf.KeyType{Name: "Arms", Type: raf.TypeMap},
				raf.KeyType{Name: "Body", Type: raf.TypeMap},
				raf.KeyType{Name: "Feet", Type: raf.TypeMap},
				raf.KeyType{Name: "Head", Type: raf.TypeMap},
				raf.KeyType{Name: "Legs", Type: raf.TypeMap},
				raf.KeyType{Name: "Ring1", Type: raf.TypeMap},
				raf.KeyType{Name: "Ring2", Type: raf.TypeMap},
			)

			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Arms) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Body) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Feet) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Head) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Legs) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Ring1) })
			b.AddBuilderFn(func(ib *raf.Builder) error { return buildItem(ib, &player.Armors.Ring2) })

			return nil
		})

		builder.AddBool(player.CanFly)
		builder.AddBool(player.CanSwim)
		builder.AddBool(player.CanTeleport)
		builder.AddStringArray(player.Friends...)
		builder.AddInt64(int64(player.Health))
		builder.AddInt64(int64(player.ID))

		// Items
		builder.AddArrayFn(raf.TypeMap, len(player.Items), func(b *raf.ArrayBuilder) error {
			innerBuilder := b.InnerBuilder()

			for i := range player.Items {
				innerBuilder.Reset()

				if err := buildItem(innerBuilder, &player.Items[i]); err != nil {
					return err
				}

				data, err := innerBuilder.Build()
				if err != nil {
					return err
				}

				b.AddRaw(data)
			}
			return nil
		})

		builder.AddInt64(int64(player.Level))
		builder.AddInt64(int64(player.Mana))
		builder.AddString(player.Name)

		// Weapon
		builder.AddBuilderFn(func(b *raf.Builder) error {
			return buildItem(b, &player.Weapon)
		})

		_, err := builder.Build()
		if err != nil {
			b.Fatal(err)
		}
	}

}

func buildItem(b *raf.Builder, item *PlayerItem) error {
	b.AddKeys(
		raf.KeyType{Name: "ID", Type: raf.TypeInt64},
		raf.KeyType{Name: "Name", Type: raf.TypeString},
		raf.KeyType{Name: "Type", Type: raf.TypeInt64},
	)

	b.AddInt64(int64(item.ID))
	b.AddString(item.Name)
	b.AddInt64(int64(item.Type))
	return nil
}

type ItemType int

const (
	ItemWeapon ItemType = iota
	ItemArmor
	ItemPotion
	ItemFood
	ItemMisc
)

type Player struct {
	ID          int          `json:"id" raf:"id"`
	Name        string       `json:"name" raf:"name"`
	Items       []PlayerItem `json:"items" raf:"items"`
	Weapon      PlayerItem   `json:"weapon" raf:"weapon"`
	Armors      PlayerArmors `json:"armors" raf:"armors"`
	Health      int          `json:"health" raf:"health"`
	Mana        int          `json:"mana" raf:"mana"`
	Level       int          `json:"level" raf:"level"`
	CanFly      bool         `json:"can_fly" raf:"can_fly"`
	CanSwim     bool         `json:"can_swim" raf:"can_swim"`
	CanTeleport bool         `json:"can_teleport" raf:"can_teleport"`

	Friends []string `json:"friends" raf:"friends"`
}

type PlayerItem struct {
	ID   int      `json:"id" raf:"id"`
	Name string   `json:"name" raf:"name"`
	Type ItemType `json:"type" raf:"type"`
}

type PlayerArmors struct {
	Head  PlayerItem `json:"head" raf:"head"`
	Body  PlayerItem `json:"body" raf:"body"`
	Legs  PlayerItem `json:"legs" raf:"legs"`
	Arms  PlayerItem `json:"arms" raf:"arms"`
	Feet  PlayerItem `json:"feet" raf:"feet"`
	Ring1 PlayerItem `json:"ring1" raf:"ring1"`
	Ring2 PlayerItem `json:"ring2" raf:"ring2"`
}

var itemNamesByType = map[ItemType][]string{
	ItemWeapon: {"Sword", "Axe", "Dagger", "Bow", "Spear"},
	ItemArmor:  {"Helm", "Chestplate", "Greaves", "Gauntlets", "Boots", "Ring"},
	ItemPotion: {"Healing Potion", "Mana Potion", "Stamina Potion", "Antidote"},
	ItemFood:   {"Bread", "Cheese", "Apple", "Jerky"},
	ItemMisc:   {"Torch", "Rope", "Gem", "Key", "Map"},
}

var playerNames = []string{
	"Arin",
	"Bryn",
	"Cora",
	"Dax",
	"Eira",
	"Finn",
	"Kael",
	"Lyra",
	"Mira",
	"Nash",
}

func randomPlayer() Player {
	level := rng.Intn(60) + 1
	items := make([]PlayerItem, rng.Intn(10)+5)

	for i := range items {
		items[i] = randomItem(ItemType(rng.Intn(int(ItemMisc) + 1)))
	}

	friends := make([]string, rng.Intn(5))
	for i := range friends {
		friends[i] = playerNames[rng.Intn(len(playerNames))]
	}

	return Player{
		ID:          rng.Intn(1_000_000) + 1,
		Name:        fmt.Sprintf("%s-%d", playerNames[rng.Intn(len(playerNames))], rng.Intn(10_000)),
		Items:       items,
		Weapon:      randomItem(ItemWeapon),
		Armors:      randomArmors(),
		Health:      rng.Intn(400) + 100,
		Mana:        rng.Intn(250),
		Level:       level,
		CanFly:      rng.Intn(10) == 0,
		CanSwim:     rng.Intn(2) == 0,
		CanTeleport: rng.Intn(20) == 0,
		Friends:     friends,
	}
}

func randomArmors() PlayerArmors {
	return PlayerArmors{
		Head:  randomNamedArmor("Helm"),
		Body:  randomNamedArmor("Chestplate"),
		Legs:  randomNamedArmor("Greaves"),
		Arms:  randomNamedArmor("Gauntlets"),
		Feet:  randomNamedArmor("Boots"),
		Ring1: randomNamedArmor("Ring"),
		Ring2: randomNamedArmor("Ring"),
	}
}

func randomNamedArmor(name string) PlayerItem {
	return PlayerItem{
		ID:   rng.Intn(1_000_000) + 1,
		Name: name,
		Type: ItemArmor,
	}
}

func randomItem(itemType ItemType) PlayerItem {
	names := itemNamesByType[itemType]

	return PlayerItem{
		ID:   rng.Intn(1_000_000) + 1,
		Name: names[rng.Intn(len(names))],
		Type: itemType,
	}
}

func generatePlayers(count int) []Player {
	records := make([]Player, count)
	for i := range records {
		records[i] = randomPlayer()
	}
	return records
}
