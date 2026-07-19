package main

func main() {
	total := 0
	for i := 0; i < 12; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 12-2 {
			break
		}
		total = total + i
	}
	if total == 40 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
