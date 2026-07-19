package main

func main() {
	text := "aaaaaaPASS\nbbbb"
	start := len("aaaaaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
