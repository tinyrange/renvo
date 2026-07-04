package main

func main() {
	values := []int{0, 1, 9}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
