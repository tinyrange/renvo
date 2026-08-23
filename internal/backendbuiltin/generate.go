// Package backendbuiltin exposes the checked-in RTG definitions without
// embedding any generated target compiler.
package backendbuiltin

//go:generate go run ./cmd/gen -definitions ../../backend/definitions -o definitions_generated.go
