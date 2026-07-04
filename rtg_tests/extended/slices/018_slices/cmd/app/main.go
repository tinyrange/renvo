package main

func main() {
	values := []int{7, 6, 3}
	values = append(values[1:2], 21)
	if len(values) == 2 && values[0]+values[1] == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
