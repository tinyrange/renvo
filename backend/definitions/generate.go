// Package definitions owns the authored built-in machine definitions and the
// checked-in Go generated from their declarative architecture contracts.
package definitions

//go:generate go run ../../internal/rtg/cmd/rtggen -kernel -package main -o ../compiler_rtg_generated_impl.go
//go:generate go run ../../internal/rtg/cmd/rtggen -production -arch aarch64 -package main -o ../compiler_aarch64_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -production -arch x86_64 -package main -o ../compiler_amd64_target_impl.go amd64.rtg
