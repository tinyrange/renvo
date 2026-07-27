// Package definitions owns the authored built-in machine definitions and the
// checked-in Go generated from their declarative architecture contracts.
package definitions

//go:generate go run ../../internal/rtg/cmd/rtggen -arch aarch64 -package main -o ../compiler_aarch64_generated_impl.go aarch64.rtg
