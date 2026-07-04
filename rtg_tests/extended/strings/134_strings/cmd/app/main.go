package main

func main() {
	text := "aaaaaaaaPASS\nb"
	start := len("aaaaaaaa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
