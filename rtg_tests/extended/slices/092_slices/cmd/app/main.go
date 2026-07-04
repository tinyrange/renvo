package main

func main() {
	values := []int{4, 2, 9}
	values = append(values[1:2], 19)
	if len(values) == 2 && values[0]+values[1] == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
