package main

func main() {
	values := map[float64]int64{0: 0}
	values[0] += -1
	if values[0] != int64(-1) {
		print("FAIL: wide map compound assignment\n")
		return
	}
	print("PASS\n")
}
