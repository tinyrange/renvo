package main

func main() {
	text := "aaaaaPASS\nb"
	start := len("aaaaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
