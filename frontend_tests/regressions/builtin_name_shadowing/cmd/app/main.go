package main

func min(left int, right int) int { return left + right }
func max(left int, right int) int { return left - right }
func clear(value *int)            { *value = 99 }

func main() {
	value := 1
	clear(&value)
	if min(2, 3) != 5 || max(7, 4) != 3 || value != 99 {
		print("FAIL")
		return
	}
	print("PASS\n")
}
