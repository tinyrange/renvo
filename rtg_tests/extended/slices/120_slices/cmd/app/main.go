package main

func main() {
	values := []int{10, 4, 3}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
