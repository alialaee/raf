package raf

import (
	"testing"

	"encoding/json"
)

type BenchUser struct {
	ID       int64    `raf:"id" json:"id"`
	Name     string   `raf:"name" json:"name"`
	IsActive bool     `raf:"is_active" json:"is_active"`
	Score    float64  `raf:"score" json:"score"`
	Roles    []string `raf:"roles" json:"roles"`
}

var benchUser = BenchUser{
	ID:       123456789,
	Name:     "Alice Smith",
	IsActive: true,
	Score:    99.99,
	Roles:    []string{"admin", "editor", "viewer"},
}

// Global to prevent optimization
var benchData []byte
var benchResult BenchUser

func BenchmarkMarshalStruct(b *testing.B) {
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchData, err = Marshal(benchUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalManual(b *testing.B) {
	builder := NewBuilder()
	var err error

	keyID := "id"
	keyIsActive := "is_active"
	keyName := "name"
	keyRoles := "roles"
	keyScore := "score"

	b.ReportAllocs()

	buf := make([]byte, 0, 1024)

	for b.Loop() {
		builder.Reset()

		// Keys must be added in sorted order
		builder.AddInt64(keyID, benchUser.ID)
		builder.AddBool(keyIsActive, benchUser.IsActive)
		builder.AddStringString(keyName, benchUser.Name)
		builder.AddStringStringArray(keyRoles, benchUser.Roles)
		builder.AddFloat64(keyScore, benchUser.Score)

		benchData, err = builder.Build(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalStruct(b *testing.B) {
	data, err := Marshal(benchUser)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var u BenchUser
		err = Unmarshal(data, &u)
		if err != nil {
			b.Fatal(err)
		}
		benchResult = u
	}
}

func BenchmarkUnmarshalManual(b *testing.B) {
	data, err := Marshal(benchUser)
	if err != nil {
		b.Fatal(err)
	}

	keyID := []byte("id")
	keyIsActive := []byte("is_active")
	keyName := []byte("name")
	keyRoles := []byte("roles")
	keyScore := []byte("score")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var u BenchUser
		block := NewBlock(data)
		if !block.Valid() {
			b.Fatal("invalid block")
		}

		if val, ok := block.Get(keyID); ok {
			u.ID = val.Int64()
		}
		if val, ok := block.Get(keyIsActive); ok {
			u.IsActive = val.Bool()
		}
		if val, ok := block.Get(keyName); ok {
			u.Name = val.String()
		}
		if val, ok := block.Get(keyRoles); ok {
			arr := val.Array()
			u.Roles = make([]string, arr.Len())
			for j := 0; j < arr.Len(); j++ {
				u.Roles[j] = string(arr.At(j))
			}
		}
		if val, ok := block.Get(keyScore); ok {
			u.Score = val.Float64()
		}
		benchResult = u
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchData, err = json.Marshal(benchUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	data, err := json.Marshal(benchUser)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var u BenchUser
		err = json.Unmarshal(data, &u)
		if err != nil {
			b.Fatal(err)
		}
		benchResult = u
	}
}
