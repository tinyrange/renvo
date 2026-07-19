package main

func main() {
	text := "aaaaaPASS\n"
	start := len("aaaaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
