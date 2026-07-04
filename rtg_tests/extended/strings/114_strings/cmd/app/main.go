package main

func main() {
	text := "aaaaaaPASS\nbb"
	start := len("aaaaaa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
