package main

func main() {
	var m map[string]int
	m = make(map[string]int)
	m["x"] = 3
	if m["x"] == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
