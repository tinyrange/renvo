package main

func main() {
	text := "PASS\nb"
	start := len("")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
