package main

func makeText() string {
	var buf []byte
	text := "xPASSy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("x")
	end := len(text) - len("y")
	if text[start:end] == "PASS" && len(text) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
