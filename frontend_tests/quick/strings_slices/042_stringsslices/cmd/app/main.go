package main

func makeText() string {
	var buf []byte
	text := "xxPASSyyy"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("xx")
	end := len(text) - len("yyy")
	if text[start:end] == "PASS" && len(text) == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
