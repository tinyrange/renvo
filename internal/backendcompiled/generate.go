// Package backendcompiled contains the checked-in ordinary-Go bundle of all
// built-in Renvo backends.
package backendcompiled

//go:generate go run ./cmd/gen -backend ../../backend -o compiler_generated.go -sources sources_generated.go
