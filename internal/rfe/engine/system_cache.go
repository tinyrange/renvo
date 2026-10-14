package engine

import "fmt"

// SystemCacheBudget selects bounded user/kernel storage independently of guest
// semantics. The zero/default preset preserves the original correctness budget.
type SystemCacheBudget struct {
	UserBlocks, UserBytes, KernelBlocks, KernelBytes int
}

func SystemCachePreset(name string) (SystemCacheBudget, error) {
	switch name {
	case "", "default":
		return SystemCacheBudget{3072, 6 << 20, 1024, 2 << 20}, nil
	case "balanced8":
		return SystemCacheBudget{2048, 4 << 20, 2048, 4 << 20}, nil
	case "balanced32":
		return SystemCacheBudget{8192, 16 << 20, 8192, 16 << 20}, nil
	case "kernel32":
		return SystemCacheBudget{4096, 8 << 20, 12288, 24 << 20}, nil
	case "balanced64":
		return SystemCacheBudget{16384, 32 << 20, 16384, 32 << 20}, nil
	default:
		return SystemCacheBudget{}, fmt.Errorf("unknown JIT cache preset %q", name)
	}
}
