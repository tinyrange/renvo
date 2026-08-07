package main

func main() {
	values := map[string]int64{"zero": 31, "one": 9}
	key := "zero"
	key, values[key] = "one", -44
	if key != "one" || values["zero"] != int64(-44) || values["one"] != int64(9) {
		print("FAIL: wide map multiple assignment\n")
		return
	}
	print("PASS\n")
}
