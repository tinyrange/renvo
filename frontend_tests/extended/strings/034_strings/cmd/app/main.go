package main

func main() {
	text := "aaaaaaaPASS\nbbbbbb"
	start := len("aaaaaaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
