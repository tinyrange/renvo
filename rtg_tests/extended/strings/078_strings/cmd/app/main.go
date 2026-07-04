package main

func main() {
	text := "aaaaaaPASS\nb"
	start := len("aaaaaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
