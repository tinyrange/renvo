package main

func main() {
	values := []int{8, 4, 7}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
