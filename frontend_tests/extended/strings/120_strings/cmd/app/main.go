package main

func main() {
	text := "aaaPASS\nb"
	start := len("aaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
