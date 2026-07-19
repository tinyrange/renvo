package main

func main() {
	values := []int{7, 7, 18}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
