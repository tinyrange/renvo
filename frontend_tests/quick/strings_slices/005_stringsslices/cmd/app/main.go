package main

func makeText() string {
	var buf []byte
	text := "PASSy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("")
	end := len(text) - len("y")
	if text[start:end] == "PASS" && len(text) == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
