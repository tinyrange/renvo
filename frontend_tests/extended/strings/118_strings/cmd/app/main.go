package main

func main() {
	text := "aPASS\nbbbbbb"
	start := len("a")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
