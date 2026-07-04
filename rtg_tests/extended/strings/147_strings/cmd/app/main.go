package main

func main() {
	text := "aaaPASS\n"
	start := len("aaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
