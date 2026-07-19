package main

func main() {
	text := "aaaaaaaPASS\n"
	start := len("aaaaaaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
