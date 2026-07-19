package main

func makeText() string {
	var buf []byte
	text := "xxPASS"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("xx")
	end := len(text) - len("")
	if text[start:end] == "PASS" && len(text) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
