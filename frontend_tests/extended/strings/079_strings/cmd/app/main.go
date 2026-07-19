package main

func main() {
	text := "aaaaaaaPASS\nbb"
	start := len("aaaaaaa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
