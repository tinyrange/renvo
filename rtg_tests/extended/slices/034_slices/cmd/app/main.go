package main

func main() {
	values := []int{1, 9, 2}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
