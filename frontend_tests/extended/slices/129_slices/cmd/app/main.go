package main

func main() {
	values := []int{8, 13, 12}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
