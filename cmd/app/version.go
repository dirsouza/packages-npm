package main

// Version and Build are injected at compile time via -ldflags:
//
//	go build -ldflags "-X main.Version=1.0.0 -X main.Build=00001" ./cmd/app
//
// Fallback values are used during local development (make run / go run).
var (
	Version = "dev"
	Build   = "00000"
)
