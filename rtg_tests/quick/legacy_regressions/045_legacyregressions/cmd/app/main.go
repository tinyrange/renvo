package main

func main() {
	a, b := map[string]int{"x": 1}, map[string]int{"y": 2}
	if a["x"] == 1 && b["y"] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
