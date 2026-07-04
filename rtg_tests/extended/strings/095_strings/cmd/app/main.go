package main

func main() {
	text := "aaaaaPASS\nbbbb"
	start := len("aaaaa")
	end := len(text) - len("bbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
