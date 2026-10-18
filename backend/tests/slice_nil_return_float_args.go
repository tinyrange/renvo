package main

type Part struct {
	PC    uint64
	Value *int
}

func search(total int, score, probability float64) []Part {
	if total == 0 {
		return nil
	}
	if probability > 0.5 {
		return []Part{{PC: uint64(total)}}
	}
	return nil
}
func appMain(args []string) int {
	for i := 0; i < 4; i++ {
		if result := search(0, 1, 1); len(result) != 0 {
			print("FAIL nil result\n")
			return 1
		}
		if result := search(7, 1, 1); len(result) != 1 || result[0].PC != 7 {
			print("FAIL nonnil result\n")
			return 1
		}
		if result := search(7, 1, 0); len(result) != 0 {
			print("FAIL second nil\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
