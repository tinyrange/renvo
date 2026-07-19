package main

func main() {
	text := "aaaaPASS\nb"
	start := len("aaaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
