package main

func main() {
	text := "aPASS\nbb"
	start := len("a")
	end := len(text) - len("bb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
