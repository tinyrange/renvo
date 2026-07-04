package main

func main() {
	text := "PASS\nbbbb"
	start := len("")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
