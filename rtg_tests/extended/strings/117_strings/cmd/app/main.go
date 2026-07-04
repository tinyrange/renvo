package main

func main() {
	text := "PASS\nbbbbb"
	start := len("")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
