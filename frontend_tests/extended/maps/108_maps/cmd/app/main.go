package main

func main() {
	m := map[string]int{"a": 7, "b": 6}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
