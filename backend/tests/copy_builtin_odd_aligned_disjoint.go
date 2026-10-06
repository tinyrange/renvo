package main

func checkCopyRange(buffer []byte, before []byte, source int, destination int, count int) bool {
	// Reuse storage across the exhaustive matrix. VM targets have bounded
	// arenas, so a fresh pair of heap buffers per case is unnecessary.
	copy(buffer, before)
	if copy(buffer[destination:destination+count], buffer[source:source+count]) != count {
		return false
	}
	for i := 0; i < len(buffer); i++ {
		expected := before[i]
		if i >= destination && i < destination+count {
			expected = before[source+i-destination]
		}
		if buffer[i] != expected {
			return false
		}
	}
	return true
}

func appMain(args []string) int {
	buffer := make([]byte, 768)
	before := make([]byte, 768)
	for i := 0; i < len(before); i++ {
		before[i] = byte(i*37 + i/7)
	}
	for count := 0; count <= 273; count++ {
		for alignment := 0; alignment < 4; alignment++ {
			source := 32 + alignment
			for distance := -7; distance <= 16; distance++ {
				if !checkCopyRange(buffer, before, source, source+distance, count) {
					return 1
				}
			}
			if !checkCopyRange(buffer, before, source, 416+alignment, count) || !checkCopyRange(buffer, before, 416+alignment, source, count) {
				return 2
			}
			if !checkCopyRange(buffer, before, source, 417+alignment, count) || !checkCopyRange(buffer, before, 417+alignment, source, count) {
				return 3
			}
		}
	}
	print("PASS\n")
	return 0
}
