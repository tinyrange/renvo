package main

func main() {
	text := "aaaPASS\nbbb"
	start := len("aaa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
