package main

func main() {
	values := []int{7, 1, 17}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
