package check

// Array lengths and constant indices belong to the target's int domain. Keep
// their magnitude in 64 bits even when the checking compiler has 32-bit ints.
func wideInt64(value wideConstant) (int64, bool) {
	if !value.ok {
		return 0, false
	}
	limit := uint64(1)<<63 - 1
	if value.negative {
		limit++
	}
	magnitude := uint64(0)
	for i := len(value.words) - 1; i >= 0; i-- {
		word := uint64(value.words[i])
		if magnitude > (limit-word)/32768 {
			return 0, false
		}
		magnitude = magnitude*32768 + word
	}
	if value.negative {
		return -int64(magnitude), true
	}
	return int64(magnitude), true
}
