package main

func main() {
	text := "aaaPASS\nbbbb"
	start := len("aaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
