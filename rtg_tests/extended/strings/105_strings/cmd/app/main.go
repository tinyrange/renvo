package main

func main() {
	text := "aaaaaaPASS\n"
	start := len("aaaaaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
