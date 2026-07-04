package main

func main() {
	values := []int{6, 12, 18}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
