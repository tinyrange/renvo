package main

func main() {
	values := []int{6, 7, 15}
	values = append(values[1:2], 19)
	if len(values) == 2 && values[0]+values[1] == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
