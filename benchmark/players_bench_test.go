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
