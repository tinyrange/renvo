package backendjit

import (
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/targetinfo"
)

func excludedFamily(descriptor rtg.TargetDescriptor) map[string]bool {
	names := []string{
		"compiler_rtg_generated_impl.go",
	}
	switch descriptor.ISA {
	case "amd64", "x86_64":
		names = append(names, "compiler_amd64_target_impl.go")
	case "386", "x86":
		names = append(names, "compiler_386_target_impl.go")
	case "aarch64", "arm64":
		names = append(names, "compiler_aarch64_impl.go")
	case "arm":
		names = append(names, "compiler_arm_impl.go")
	case "wasm32", "vm32":
		names = append(names, "compiler_wasm32_impl.go")
	}
	excluded := make(map[string]bool, len(names))
	for _, name := range names {
		excluded[name] = true
	}
	return excluded
}

func representativeTarget(descriptor rtg.TargetDescriptor) string {
	return targetinfo.Representative(descriptor.OS, descriptor.ISA, descriptor.WordBits, descriptor.Capabilities)
}
