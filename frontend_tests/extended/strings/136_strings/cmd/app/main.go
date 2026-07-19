package main

func main() {
	text := "aPASS\nbbb"
	start := len("a")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
