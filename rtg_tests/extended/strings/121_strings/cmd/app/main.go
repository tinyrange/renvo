package main

func main() {
	text := "aaaaPASS\nbb"
	start := len("aaaa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
