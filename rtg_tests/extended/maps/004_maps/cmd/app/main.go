package main

func main() {
	m := map[string]int{"a": 5, "b": 6}
	m["a"] = m["a"] + m["b"]
	corpusOK := false
	if m["a"] == 11 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
