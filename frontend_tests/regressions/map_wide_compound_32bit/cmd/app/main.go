package main

func main() {
	values := map[float64]int64{2: 0}
	values[2] += -1
	if values[2] != int64(-1) {
		print("FAIL: wide map compound assignment\n")
		return
	}
	print("PASS\n")
}
