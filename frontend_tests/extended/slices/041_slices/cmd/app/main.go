package main

func main() {
	values := []int{8, 3, 9}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
