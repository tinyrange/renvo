package main

func main() {
	text := "aaPASS\nb"
	start := len("aa")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
