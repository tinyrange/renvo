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
	corpusOK := false
	if total == 31 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
