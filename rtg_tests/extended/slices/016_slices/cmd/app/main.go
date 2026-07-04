package main

func main() {
	values := []int{5, 4, 18}
	values = append(values[1:2], 19)
	if len(values) == 2 && values[0]+values[1] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
