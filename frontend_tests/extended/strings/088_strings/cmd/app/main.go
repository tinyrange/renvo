package main

func main() {
	text := "aaaaaaaPASS\nbbbb"
	start := len("aaaaaaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
