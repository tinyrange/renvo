package main

func main() {
	values := []int{10, 3, 5}
	values = append(values[1:2], 19)
	if len(values) == 2 && values[0]+values[1] == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
