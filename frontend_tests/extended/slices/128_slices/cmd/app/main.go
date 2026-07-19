package main

func main() {
	values := []int{7, 12, 11}
	values = append(values[1:2], 17)
	if len(values) == 2 && values[0]+values[1] == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
