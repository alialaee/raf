package main

import (
	"fmt"
	"testing"
)

var apiResponses []APIResponse = generateAPIResponses(1000)

func BenchmarkAllMarshals_APIResponse(b *testing.B) {
	benchmarkAllMarshals(b, apiResponses)
}

func BenchmarkAllUnmarshals_APIResponse(b *testing.B) {
	benchmarkAllUnmarshals(b, apiResponses)
}

type APIData struct {
	Items      []APIItem     `json:"items" raf:"items"`
	Pagination APIPagination `json:"pagination" raf:"pagination"`
}

type APIItem struct {
	ID             string     `json:"id" raf:"id"`
	Username       string     `json:"username" raf:"username"`
	Profile        APIProfile `json:"profile" raf:"profile"`
	Tags           []string   `json:"tags" raf:"tags"`
	AccountBalance float64    `json:"account_balance" raf:"account_balance"`
	IsActive       bool       `json:"is_active" raf:"is_active"`
	LastLogin      *string    `json:"last_login,omitempty" raf:"last_login,omitempty"`
}

type APIProfile struct {
	FirstName string  `json:"first_name" raf:"first_name"`
	LastName  string  `json:"last_name" raf:"last_name"`
	AvatarURL *string `json:"avatar_url,omitempty" raf:"avatar_url,omitempty"`
}

type APIPagination struct {
	TotalItems  int `json:"total_items" raf:"total_items"`
	PageSize    int `json:"page_size" raf:"page_size"`
	CurrentPage int `json:"current_page" raf:"current_page"`
	TotalPages  int `json:"total_pages" raf:"total_pages"`
}

type APIResponse struct {
	Status    string  `json:"status" raf:"status"`
	Data      APIData `json:"data" raf:"data"`
	RequestID string  `json:"request_id" raf:"request_id"`
}

var apiStatuses = []string{"success", "error", "pending"}

func generateAPIResponses(count int) []APIResponse {
	responses := make([]APIResponse, count)
	for i := range responses {
		responses[i] = randomAPIResponse()
	}
	return responses
}

func randomAPIResponse() APIResponse {
	status := apiStatuses[rng.Intn(len(apiStatuses))]

	return APIResponse{
		Status:    status,
		Data:      randomAPIData(),
		RequestID: fmt.Sprintf("req_%d", rng.Intn(1_000_000)),
	}
}

func randomAPIData() APIData {
	numItems := rng.Intn(10) + 1
	items := make([]APIItem, numItems)
	for i := range items {
		items[i] = randomAPIItem()
	}

	return APIData{
		Items: items,
		Pagination: APIPagination{
			TotalItems:  numItems,
			PageSize:    numItems,
			CurrentPage: 1,
			TotalPages:  1,
		},
	}
}

func randomAPIItem() APIItem {
	numTags := rng.Intn(5)
	tags := make([]string, numTags)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag%d", rng.Intn(100))
	}

	var lastLogin *string
	if rng.Intn(2) == 0 {
		loginTime := fmt.Sprintf("2024-06-%02dT%02d:%02d:%02dZ", rng.Intn(30)+1, rng.Intn(24), rng.Intn(60), rng.Intn(60))
		lastLogin = &loginTime
	}

	var avatarURL *string
	if rng.Intn(2) == 0 {
		url := fmt.Sprintf("https://example.com/avatar/%d.png", rng.Intn(1_000_000))
		avatarURL = &url
	}

	return APIItem{
		ID:             fmt.Sprintf("user_%d", rng.Intn(1_000_000)),
		Username:       fmt.Sprintf("user%d", rng.Intn(1_000_000)),
		Profile:        APIProfile{FirstName: fmt.Sprintf("First%d", rng.Intn(100)), LastName: fmt.Sprintf("Last%d", rng.Intn(100)), AvatarURL: avatarURL},
		Tags:           tags,
		AccountBalance: float64(rng.Intn(10000)) / 100,
		IsActive:       rng.Intn(2) == 0,
		LastLogin:      lastLogin,
	}
}
