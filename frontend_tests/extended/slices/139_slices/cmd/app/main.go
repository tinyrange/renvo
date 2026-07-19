package main

func main() {
	values := []int{7, 10, 5}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
