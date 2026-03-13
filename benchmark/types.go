package main

import (
	"fmt"
	"math/rand"
)

var rngSource = rand.NewSource(42)
var rng = rand.New(rngSource)

type ItemType int

const (
	ItemWeapon ItemType = iota
	ItemArmor
	ItemPotion
	ItemFood
	ItemMisc
)

type Player struct {
	ID          int    `json:"id" raf:"id"`
	Name        string `json:"name" raf:"name"`
	Items       []Item `json:"items" raf:"items"`
	Weapon      Item   `json:"weapon" raf:"weapon"`
	Armors      Armors `json:"armors" raf:"armors"`
	Health      int    `json:"health" raf:"health"`
	Mana        int    `json:"mana" raf:"mana"`
	Level       int    `json:"level" raf:"level"`
	CanFly      bool   `json:"can_fly" raf:"can_fly"`
	CanSwim     bool   `json:"can_swim" raf:"can_swim"`
	CanTeleport bool   `json:"can_teleport" raf:"can_teleport"`

	Friends []string `json:"friends" raf:"friends"`
}

type Item struct {
	ID   int      `json:"id" raf:"id"`
	Name string   `json:"name" raf:"name"`
	Type ItemType `json:"type" raf:"type"`
}

type Armors struct {
	Head  Item `json:"head" raf:"head"`
	Body  Item `json:"body" raf:"body"`
	Legs  Item `json:"legs" raf:"legs"`
	Arms  Item `json:"arms" raf:"arms"`
	Feet  Item `json:"feet" raf:"feet"`
	Ring1 Item `json:"ring1" raf:"ring1"`
	Ring2 Item `json:"ring2" raf:"ring2"`
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
	items := make([]Item, rng.Intn(10)+5)

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

func randomArmors() Armors {
	return Armors{
		Head:  randomNamedArmor("Helm"),
		Body:  randomNamedArmor("Chestplate"),
		Legs:  randomNamedArmor("Greaves"),
		Arms:  randomNamedArmor("Gauntlets"),
		Feet:  randomNamedArmor("Boots"),
		Ring1: randomNamedArmor("Ring"),
		Ring2: randomNamedArmor("Ring"),
	}
}

func randomNamedArmor(name string) Item {
	return Item{
		ID:   rng.Intn(1_000_000) + 1,
		Name: name,
		Type: ItemArmor,
	}
}

func randomItem(itemType ItemType) Item {
	names := itemNamesByType[itemType]

	return Item{
		ID:   rng.Intn(1_000_000) + 1,
		Name: names[rng.Intn(len(names))],
		Type: itemType,
	}
}

func createRecords(count int) []Player {
	records := make([]Player, count)
	for i := range records {
		records[i] = randomPlayer()
	}
	return records
}
