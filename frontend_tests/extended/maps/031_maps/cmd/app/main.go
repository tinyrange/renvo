package main

func main() {
	m := map[string]int{"a": 15, "b": 7}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
