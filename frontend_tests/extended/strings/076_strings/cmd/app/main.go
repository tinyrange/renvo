package main

func main() {
	text := "aaaaPASS\nbbbbbb"
	start := len("aaaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
