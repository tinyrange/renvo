package main

func main() {
	text := "aaPASS\nbbbb"
	start := len("aa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
