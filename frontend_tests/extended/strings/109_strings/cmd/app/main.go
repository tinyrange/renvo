package main

func main() {
	text := "aPASS\nbbbb"
	start := len("a")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
