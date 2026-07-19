package main

func main() {
	values := []int{0, 11, 5}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
