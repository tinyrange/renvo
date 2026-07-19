package main

func main() {
	text := "aaaaaaPASS\nbbbbb"
	start := len("aaaaaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
