package main

func main() {
	text := "aaPASS\nbbbbbb"
	start := len("aa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
