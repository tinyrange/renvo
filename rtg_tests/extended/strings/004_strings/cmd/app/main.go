package main

func main() {
	text := "aaaaPASS\nbbbb"
	start := len("aaaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
