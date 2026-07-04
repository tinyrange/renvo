package main

func main() {
	text := "aaaaaaaaPASS\nbbbbbb"
	start := len("aaaaaaaa")
	end := len(text) - len("bbbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
