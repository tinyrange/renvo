package main

func main() {
	text := "aaaaPASS\nbbb"
	start := len("aaaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
