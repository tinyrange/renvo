package main

func main() {
	text := "aaaaPASS\nbbbb"
	start := len("aaaa")
	end := len(text) - len("bbbb")
	corpusOK := false
	if text[start:end] == "PASS\n" {
		corpusOK = true
	}
	if corpusOK {
		print(text[start:end])
		return
	}

	print("FAIL\n")
}
