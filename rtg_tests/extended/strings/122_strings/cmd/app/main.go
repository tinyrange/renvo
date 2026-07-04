package main

func main() {
	text := "aaaaaPASS\nbbb"
	start := len("aaaaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
