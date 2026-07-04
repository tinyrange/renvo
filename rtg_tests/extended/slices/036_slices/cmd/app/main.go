package main

func main() {
	values := []int{3, 11, 4}
	values = append(values[1:2], 20)
	if len(values) == 2 && values[0]+values[1] == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
