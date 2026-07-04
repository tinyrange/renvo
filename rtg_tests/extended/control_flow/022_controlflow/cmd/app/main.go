package main

func main() {
	total := 0
	for i := 0; i < 8; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 8-2 {
			break
		}
		total = total + i
	}
	if total == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
