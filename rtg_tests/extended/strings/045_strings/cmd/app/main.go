package main

func main() {
	text := "PASS\nbbb"
	start := len("")
	end := len(text) - len("bbb")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
