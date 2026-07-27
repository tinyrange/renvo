package backendjit

import "renvo.dev/internal/rtg"

func excludedFamily(descriptor rtg.TargetDescriptor) map[string]bool {
	names := []string{
		"compiler_rtg_generated_impl.go",
		"compiler_amd64_generated_impl.go",
		"compiler_aarch64_generated_impl.go",
	}
	switch descriptor.ISA {
	case "amd64", "x86_64":
		// The shared backend kernel still owns target-neutral lowering and the
		// stable amd64 adapter surface while the generated definition owns the
		// encoder. Keep those kernel files until their adapters have all moved
		// onto the generated operation contract.
	case "386", "x86":
		names = append(names,
			"compiler_386_impl.go", "compiler_386_target_impl.go",
			"compiler_linux_386_impl.go", "compiler_windows_386_impl.go",
		)
	case "aarch64", "arm64":
		names = append(names,
			"compiler_aarch64_impl.go", "compiler_aarch64_target_impl.go",
			"compiler_linux_aarch64_impl.go", "compiler_darwin_arm64_impl.go",
			"compiler_windows_arm64_impl.go",
		)
	case "arm":
		names = append(names, "compiler_arm_impl.go", "compiler_linux_arm_impl.go")
	case "wasm32", "vm32":
		names = append(names, "compiler_wasm32_impl.go", "compiler_wasi_wasm32_impl.go")
	}
	excluded := make(map[string]bool, len(names))
	for _, name := range names {
		excluded[name] = true
	}
	return excluded
}

func representativeTarget(descriptor rtg.TargetDescriptor) string {
	if descriptor.OS == "windows" {
		if descriptor.ISA == "aarch64" || descriptor.ISA == "arm64" {
			return "windows/arm64"
		}
		if descriptor.WordBits == 32 {
			return "windows/386"
		}
		return "windows/amd64"
	}
	if descriptor.OS == "darwin" {
		return "darwin/arm64"
	}
	if descriptor.OS == "wasi" || descriptor.OS == "browser" {
		return "wasi/wasm32"
	}
	if descriptor.OS == "vm" {
		return "vm/vm32"
	}
	if hasValue(descriptor.Capabilities, "kernel_module") {
		return "linux-kernel/amd64"
	}
	if descriptor.ISA == "aarch64" || descriptor.ISA == "arm64" {
		return "linux/aarch64"
	}
	if descriptor.ISA == "arm" {
		return "linux/arm"
	}
	if descriptor.WordBits == 32 {
		return "linux/386"
	}
	return "linux/amd64"
}

func hasValue(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
