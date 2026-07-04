package main

func main() {
	total := 0
	for i := 0; i < 7; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 7-2 {
			break
		}
		total = total + i
	}
	if total == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
