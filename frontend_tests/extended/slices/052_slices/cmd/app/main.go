package main

func main() {
	values := []int{8, 1, 3}
	values = append(values[1:2], 17)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
