package main

func main() {
	k := "a"
	m := map[string]int{k: 2}
	if m["a"] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
