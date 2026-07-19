package main

func main() {
	text := "aaaPASS\nbb"
	start := len("aaa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
