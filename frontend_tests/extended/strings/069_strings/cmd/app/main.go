package main

func main() {
	text := "aaaaaaPASS\nbbbbbb"
	start := len("aaaaaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
