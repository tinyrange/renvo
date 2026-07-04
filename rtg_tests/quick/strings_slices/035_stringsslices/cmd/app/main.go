package main

func makeText() string {
	var buf []byte
	text := "PASSyy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("")
	end := len(text) - len("yy")
	if text[start:end] == "PASS" && len(text) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
