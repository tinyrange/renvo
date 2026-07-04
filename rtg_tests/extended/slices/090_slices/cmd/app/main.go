package main

func main() {
	values := []int{2, 13, 7}
	values = append(values[1:2], 17)
	if len(values) == 2 && values[0]+values[1] == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
