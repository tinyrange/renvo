package main

func main() {
	text := "aaPASS\nbbb"
	start := len("aa")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
