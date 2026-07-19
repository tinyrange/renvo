package main

func main() {
	values := []int{8, 5, 15}
	values = append(values[1:2], 14)
	if len(values) == 2 && values[0]+values[1] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
