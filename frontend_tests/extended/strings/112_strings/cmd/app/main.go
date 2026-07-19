package main

func main() {
	text := "aaaaPASS\n"
	start := len("aaaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
