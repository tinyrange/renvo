package main

func main() {
	values := []int{7, 2, 8}
	values = append(values[1:2], 5)
	if len(values) == 2 && values[0]+values[1] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
