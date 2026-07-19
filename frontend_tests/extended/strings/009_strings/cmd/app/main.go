package main

func main() {
	text := "PASS\nbb"
	start := len("")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
