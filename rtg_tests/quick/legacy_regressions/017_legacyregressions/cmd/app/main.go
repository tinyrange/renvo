package main

func main() {
	m := map[string]int{"a": 1}
	got := ""
	for k := range m {
		got = k
	}
	if got == "a" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
