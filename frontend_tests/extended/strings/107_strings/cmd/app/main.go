package main

func main() {
	text := "aaaaaaaaPASS\nbb"
	start := len("aaaaaaaa")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
