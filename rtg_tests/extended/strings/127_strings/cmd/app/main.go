package main

func main() {
	text := "aPASS\nb"
	start := len("a")
	end := len(text) - len("b")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
