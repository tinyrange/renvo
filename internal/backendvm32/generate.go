// Package backendvm32 contains the fixed VM32 seed backend used to bootstrap
// target-specialized compilers.
package backendvm32

// Keep the shared compiler kernel and VM32 encoder, but replace every native
// or prepared-target emitter body with an unreachable stub. The fixed target
// setting makes those stubs defensive assertions rather than selectable
// backend implementations.
//go:generate go run ../backendcompiled/cmd/gen -backend ../../backend -o compiler_generated.go -sources sources_generated.go -package backendvm32 -fixed-target renvoTargetVM32 -stub-sources compiler_pe_impl.go,compiler_windows_impl.go,compiler_amd64_impl.go,compiler_amd64_target_impl.go,compiler_386_impl.go,compiler_386_target_impl.go,compiler_aarch64_impl.go,compiler_aarch64_target_impl.go,compiler_arm_impl.go,compiler_linux_amd64_impl.go,compiler_freebsd_amd64_impl.go,compiler_openbsd_amd64_impl.go,compiler_netbsd_amd64_impl.go,compiler_linux_amd64_object_impl.go,compiler_windows_amd64_impl.go,compiler_linux_kernel_amd64_target_impl.go,compiler_linux_kernel_amd64_impl.go,compiler_linux_386_impl.go,compiler_windows_386_impl.go,compiler_linux_aarch64_impl.go,compiler_windows_arm64_impl.go,compiler_linux_arm_impl.go,compiler_darwin_arm64_impl.go -stub-functions compileWasiWasm32,compileWasiWasm32Arena,renvoTryCompileWasiWasm32,renvoWasm32Image
