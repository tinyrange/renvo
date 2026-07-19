package main

func main() {
	text := "PASS\nbbbbbb"
	start := len("")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
