package main

func main() {
	text := "aaaaaaaPASS\nb"
	start := len("aaaaaaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
