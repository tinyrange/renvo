package main

func main() {
	text := "aaaaaaaaPASS\nbbb"
	start := len("aaaaaaaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
