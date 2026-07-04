package main

func main() {
	text := "aaaaaaaaPASS\nbbbbb"
	start := len("aaaaaaaa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
