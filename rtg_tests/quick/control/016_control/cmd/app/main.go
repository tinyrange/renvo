package main

func score(limit int) int {
	total := 0
	for i := 0; i < limit; i++ {
		if i%3 == 0 {
			total = total + i*2
		} else if i%3 == 1 {
			total = total + i + 4
		} else {
			total = total - i
		}
	}
	return total
}

func main() {
	if score(8) == 35 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
