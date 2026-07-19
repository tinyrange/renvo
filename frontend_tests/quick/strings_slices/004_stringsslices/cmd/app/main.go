package main

func makeText() string {
	var buf []byte
	text := "xxxxPASS"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("xxxx")
	end := len(text) - len("")
	corpusOK := false
	if text[start:end] == "PASS" && len(text) == 8 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
