package main

func makeText() string {
	var buf []byte
	text := "xxxxPASSyyyy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("xxxx")
	end := len(text) - len("yyyy")
	if text[start:end] == "PASS" && len(text) == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
