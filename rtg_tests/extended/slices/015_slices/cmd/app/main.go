package main

func main() {
	values := []int{4, 3, 17}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
