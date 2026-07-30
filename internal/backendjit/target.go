package backendjit

import "renvo.dev/internal/rtg"

func excludedFamily(descriptor rtg.TargetDescriptor) map[string]bool {
	_ = descriptor
	names := []string{
		"compiler_rtg_generated_impl.go",
		"compiler_rtg_inactive_impl.go",
	}
	excluded := make(map[string]bool, len(names))
	for _, name := range names {
		excluded[name] = true
	}
	return excluded
}
