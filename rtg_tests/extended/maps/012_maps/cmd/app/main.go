package main

func main() {
	m := map[string]int{"a": 13, "b": 14}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
