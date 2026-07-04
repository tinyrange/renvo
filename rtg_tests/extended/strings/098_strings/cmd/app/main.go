package main

func main() {
	text := "aaaaaaaaPASS\n"
	start := len("aaaaaaaa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
