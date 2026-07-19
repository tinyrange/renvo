package main

func main() {
	text := "aaaaaaaPASS\nbbb"
	start := len("aaaaaaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
