package m2

import (
	"encoding/json"
	"testing"
)

type BenchUser struct {
	ID       int64          `raf:"id" json:"id"`
	Name     string         `raf:"name" json:"name"`
	IsActive bool           `raf:"is_active" json:"is_active"`
	Score    float64        `raf:"score" json:"score"`
	Roles    []string       `raf:"roles" json:"roles"`
	Inner    BenchUserInner `raf:"inner" json:"inner"`
}

type BenchUserInner struct {
	UserID   int64         `raf:"user_id" json:"user_id"`
	Username string        `raf:"username" json:"username"`
	Auth     BenchUserAuth `raf:"auth" json:"auth"`
}

type BenchUserAuth struct {
	Password  string `raf:"password" json:"password"`
	AuthToken string `raf:"auth_token" json:"auth_token"`
}

var benchUser = BenchUser{
	ID:       123456789,
	Name:     "Ali Alaee",
	IsActive: true,
	Score:    99.99,
	Roles:    []string{"admin", "editor", "viewer"},
	Inner: BenchUserInner{
		UserID:   987654321,
		Username: "ali.alaee",
		Auth: BenchUserAuth{
			Password:  "password123",
			AuthToken: "token123",
		},
	},
}

func BenchmarkMarshal(b *testing.B) {
	var err error
	marshaler := NewMarshaler()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err = marshaler.Marshal(benchUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	var err error

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err = json.Marshal(benchUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}
