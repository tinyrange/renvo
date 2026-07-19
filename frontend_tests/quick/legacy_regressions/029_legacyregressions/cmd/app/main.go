package main

func main() {
	total := 0
	for _, v := range [1]int{7} {
		total += v
	}
	if total == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
