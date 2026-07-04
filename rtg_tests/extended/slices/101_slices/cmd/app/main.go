package main

func main() {
	values := []int{2, 11, 18}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
