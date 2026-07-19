package main

func main() {
	text := "aaaPASS\nbbbbbb"
	start := len("aaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
