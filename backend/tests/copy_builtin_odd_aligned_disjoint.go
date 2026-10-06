package main

func checkCopyRange(source int, destination int, count int) bool {
	buffer := make([]byte, 768)
	before := make([]byte, 768)
	for i := 0; i < len(buffer); i++ {
		value := byte(i*37 + i/7)
		buffer[i] = value
		before[i] = value
	}
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
	for count := 0; count <= 273; count++ {
		for alignment := 0; alignment < 4; alignment++ {
			source := 32 + alignment
			for distance := -7; distance <= 16; distance++ {
				if !checkCopyRange(source, source+distance, count) {
					return 1
				}
			}
			if !checkCopyRange(source, 416+alignment, count) || !checkCopyRange(416+alignment, source, count) {
				return 2
			}
			if !checkCopyRange(source, 417+alignment, count) || !checkCopyRange(417+alignment, source, count) {
				return 3
			}
		}
	}
	print("PASS\n")
	return 0
}
