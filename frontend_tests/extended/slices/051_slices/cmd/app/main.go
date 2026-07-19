package main

func main() {
	values := []int{7, 13, 2}
	values = append(values[1:2], 16)
	if len(values) == 2 && values[0]+values[1] == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
