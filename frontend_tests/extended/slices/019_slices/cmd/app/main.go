package main

func main() {
	values := []int{8, 7, 4}
	values = append(values[1:2], 3)
	if len(values) == 2 && values[0]+values[1] == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
