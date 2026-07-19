package main

func main() {
	text := "aPASS\nbbbbb"
	start := len("a")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
