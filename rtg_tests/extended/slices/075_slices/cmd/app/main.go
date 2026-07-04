package main

func main() {
	values := []int{9, 11, 9}
	values = append(values[1:2], 21)
	if len(values) == 2 && values[0]+values[1] == 32 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
