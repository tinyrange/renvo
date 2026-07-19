package main

func main() {
	m := map[string]int{"a": 4, "b": 9}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
