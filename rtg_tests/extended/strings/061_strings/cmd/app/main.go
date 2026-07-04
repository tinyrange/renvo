package main

func main() {
	text := "aaaaaaaPASS\nbbbbb"
	start := len("aaaaaaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
