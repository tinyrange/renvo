package main

func main() {
	text := "aaPASS\nbbbbb"
	start := len("aa")
	end := len(text) - len("bbbbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
