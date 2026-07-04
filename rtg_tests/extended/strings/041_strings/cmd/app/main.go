package main

func main() {
	text := "aaaaaPASS\nbbbbbb"
	start := len("aaaaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
