package main

func main() {
	text := "aaaPASS\nbbb"
	start := len("aaa")
	end := len(text) - len("bbb")
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if text[start:end] == "PASS\n" {
			print(text[start:end])
			return
		}
	}

	print("FAIL\n")
}
