package main

func main() {
	text := "aPASS\n"
	start := len("a")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
