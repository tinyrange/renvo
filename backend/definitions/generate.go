// Package definitions owns the authored built-in machine definitions and the
// checked-in Go generated from their declarative architecture contracts.
package definitions

//go:generate go run ../../internal/rtg/cmd/rtggen -kernel -package main -o ../compiler_rtg_generated_impl.go
//go:generate go run ../../internal/rtg/cmd/rtggen -inactive-kernel -package main -o ../compiler_rtg_inactive_impl.go
//go:generate go run ../../internal/rtg/cmd/rtggen -algorithms -arch aarch64 -package main -o ../compiler_aarch64_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -contract -arch aarch64 -package main -o ../rtg_aarch64_contract_generated.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -algorithms -arch x86_64 -package main -o ../compiler_amd64_target_impl.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -contract -arch x86_64 -package main -o ../rtg_amd64_contract_generated.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -algorithms -arch x86_32 -package main -o ../compiler_386_target_impl.go 386.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -contract -arch x86_32 -package main -o ../rtg_386_contract_generated.go 386.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -algorithms -arch arm -package main -o ../compiler_arm_impl.go arm.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -contract -arch arm -package main -o ../rtg_arm_contract_generated.go arm.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -algorithms -arch vm32 -package main -o ../compiler_wasm32_impl.go wasm32.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -contract -arch vm32 -package main -o ../rtg_vm32_contract_generated.go wasm32.rtg
