package main

func main() {
	values := []int{7, 8, 9}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
