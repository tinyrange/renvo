package main

func main() {
	values := []int{0, 7, 10}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
