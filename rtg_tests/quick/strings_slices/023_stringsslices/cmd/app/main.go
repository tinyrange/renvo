package main

func makeText() string {
	var buf []byte
	text := "xxxPASSyyyy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("xxx")
	end := len(text) - len("yyyy")
	if text[start:end] == "PASS" && len(text) == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
