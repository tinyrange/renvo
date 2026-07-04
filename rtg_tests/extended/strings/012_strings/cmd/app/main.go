package main

func main() {
	text := "aaaPASS\nbbbbb"
	start := len("aaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
