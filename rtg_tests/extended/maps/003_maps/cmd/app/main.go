package main

func main() {
	m := map[string]int{"a": 4, "b": 5}
	m["a"] = m["a"] + m["b"]
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if m["a"] == 9 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
