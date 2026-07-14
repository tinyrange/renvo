package main

func main() {
	m := map[string]int{"a": 2, "b": 3}
	m["a"] = m["a"] + m["b"]
	corpusOK := m["a"] == 5
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
