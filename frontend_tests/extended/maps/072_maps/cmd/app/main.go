package main

func main() {
	m := map[string]int{"a": 5, "b": 9}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
