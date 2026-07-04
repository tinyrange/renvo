package main

func main() {
	values := []int{6, 8, 6}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
