package main

func main() {
	text := "aaaaaPASS\nbbbbb"
	start := len("aaaaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
