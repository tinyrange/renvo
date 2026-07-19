package main

func main() {
	total := 0
	for i := 0; i < 15; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 15-2 {
			break
		}
		total = total + i
	}
	if total == 76 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
