package driver

// A zero internal limit means unlimited. User-specified limits must be positive.
func parseHeapLimit(value string) (int, bool) {
	n := len(value)
	scale := uint64(1)
	if n >= 3 && value[n-2:] == "iB" {
		switch value[n-3] {
		case 'K':
			scale = 1024
		case 'M':
			scale = 1048576
		case 'G':
			scale = 1073741824
		case 'T':
			scale = 1099511627776
		default:
			return 0, false
		}
		n -= 3
	}
	if n == 0 {
		return 0, false
	}
	var v uint64
	max := uint64(9223372036854775807) / scale
	for i := 0; i < n; i++ {
		c := value[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		d := uint64(c - '0')
		if v > max/10 || v == max/10 && d > max%10 {
			return 0, false
		}
		v = v*10 + d
	}
	v *= scale
	result := int(v)
	return result, v > 0 && result > 0 && uint64(result) == v
}

func allocationBudget(options Options) int {
	if options.MemoryStrategy == "desktop" {
		return options.HeapLimit
	}
	return backendArenaSize(options.Target, options.Tags, options.ArenaSize, options.Mode)
}
