package main

func main() {
	total := 0
	for i := 0; i < 10; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 10-2 {
			break
		}
		total = total + i
	}
	if total == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
