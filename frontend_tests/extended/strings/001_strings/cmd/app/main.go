package main

func main() {
	text := "aPASS\nb"
	start := len("a")
	end := len(text) - len("b")
	corpusOK := text[start:end] == "PASS\n"
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print(text[start:end])

}
