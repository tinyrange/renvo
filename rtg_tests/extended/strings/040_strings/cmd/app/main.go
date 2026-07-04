package main

func main() {
	text := "aaaaPASS\nbbbbb"
	start := len("aaaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
