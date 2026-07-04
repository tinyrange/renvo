package main

func main() {
	values := []int{3, 7, 9}
	values = append(values[1:2], 4)
	if len(values) == 2 && values[0]+values[1] == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
