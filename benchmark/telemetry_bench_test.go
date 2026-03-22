package main

import (
	"fmt"
	"testing"
)

var telemetryEvents = generateTelemetryEvents(1000)

func BenchmarkAllMarshals_TelemetryEvent(b *testing.B) {
	benchmarkAllMarshals(b, telemetryEvents)
}

func BenchmarkAllUnmarshals_TelemetryEvent(b *testing.B) {
	benchmarkAllUnmarshals(b, telemetryEvents)
}

func generateTelemetryEvents(n int) []TelemetryEvent {
	events := make([]TelemetryEvent, n)
	for i := range events {
		events[i] = randomTelemetryEvent()
	}
	return events
}

type TelemetryEvent struct {
	Timestamp    float64      `json:"timestamp" raf:"timestamp"`
	Level        string       `json:"level" raf:"level"`
	Service      string       `json:"service" raf:"service"`
	TraceContext TraceContext `json:"trace_context" raf:"trace_context"`
	Event        string       `json:"event" raf:"event"`
	Host         Host         `json:"host" raf:"host"`
}

type TraceContext struct {
	TraceID string `json:"trace_id" raf:"trace_id"`
	SpanID  string `json:"span_id" raf:"span_id"`
}

type Host struct {
	ID     string `json:"id" raf:"id"`
	Region string `json:"region" raf:"region"`
	IP     string `json:"ip" raf:"ip"`
}

var logLevels = []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
var regions = []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

func randomTelemetryEvent() TelemetryEvent {
	return TelemetryEvent{
		Timestamp:    rng.Float64() * 1e10,
		Level:        logLevels[rng.Intn(len(logLevels))],
		Service:      fmt.Sprintf("service%d", rng.Intn(100)),
		TraceContext: randomTraceContext(),
		Event:        fmt.Sprintf("event%d", rng.Intn(1000)),
		Host:         randomHost(),
	}
}

func randomTraceContext() TraceContext {
	return TraceContext{
		TraceID: fmt.Sprintf("%032x", rng.Int63()),
		SpanID:  fmt.Sprintf("%016x", rng.Int63()),
	}
}

func randomHost() Host {
	return Host{
		ID:     fmt.Sprintf("node-%s-%d", regions[rng.Intn(len(regions))], rng.Intn(10)+1),
		Region: regions[rng.Intn(len(regions))],
		IP:     fmt.Sprintf("10.%d.%d.%d", rng.Intn(256), rng.Intn(256), rng.Intn(256)),
	}
}
