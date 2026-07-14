package main

func main() {
	m := map[string]int{"a": 3, "b": 4}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 7 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
