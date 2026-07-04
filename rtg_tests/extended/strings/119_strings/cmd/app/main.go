package main

func main() {
	text := "aaPASS\n"
	start := len("aa")
	end := len(text) - len("")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
