// Package definitions owns the checked-in generated built-in backend sources.
package definitions

//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 1 -t linux/aarch64 -o ../compiler_aarch64_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 2 -t linux/aarch64 -o ../compiler_aarch64_target_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 3 -t linux/aarch64 -o ../compiler_linux_aarch64_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 4 -t darwin/arm64 -o ../compiler_darwin_arm64_impl.go aarch64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 5 -t windows/arm64 -o ../compiler_windows_arm64_impl.go aarch64.rtg

//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 0 -t linux/amd64 -o ../compiler_amd64_impl.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 1 -t linux/amd64 -o ../compiler_amd64_target_impl.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 2 -t linux/amd64 -o ../compiler_linux_amd64_impl.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 3 -t windows/amd64 -o ../compiler_windows_amd64_impl.go amd64.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 4 -t linux-kernel/amd64 -o ../compiler_linux_kernel_amd64_impl.go amd64.rtg

//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 0 -t linux/386 -o ../compiler_386_impl.go 386.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 1 -t linux/386 -o ../compiler_386_target_impl.go 386.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 2 -t linux/386 -o ../compiler_linux_386_impl.go 386.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 3 -t windows/386 -o ../compiler_windows_386_impl.go 386.rtg

//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 0 -t linux/arm -o ../compiler_arm_impl.go arm.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 1 -t linux/arm -o ../compiler_linux_arm_impl.go arm.rtg

//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 0 -t wasi/wasm32 -o ../compiler_wasm32_impl.go wasm32.rtg
//go:generate go run ../../internal/rtg/cmd/rtggen -builtin -package main -go-block 1 -t wasi/wasm32 -o ../compiler_wasi_wasm32_impl.go wasm32.rtg
