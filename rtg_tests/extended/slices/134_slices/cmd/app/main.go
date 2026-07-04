package main

func main() {
	values := []int{2, 5, 17}
	values = append(values[1:2], 4)
	if len(values) == 2 && values[0]+values[1] == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
