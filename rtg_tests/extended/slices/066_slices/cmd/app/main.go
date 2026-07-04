package main

func main() {
	values := []int{0, 2, 17}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
