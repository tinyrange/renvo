package main

func main() {
	text := "aaaaaaPASS\nbbb"
	start := len("aaaaaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
