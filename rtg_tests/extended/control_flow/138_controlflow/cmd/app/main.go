package main

func main() {
	total := 0
	for i := 0; i < 14; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 14-2 {
			break
		}
		total = total + i
	}
	if total == 63 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
