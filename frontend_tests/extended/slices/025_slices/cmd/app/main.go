package main

func main() {
	values := []int{3, 13, 10}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
