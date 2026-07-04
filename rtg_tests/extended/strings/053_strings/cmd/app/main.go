package main

func main() {
	text := "aaaaaaaaPASS\nbbbb"
	start := len("aaaaaaaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
