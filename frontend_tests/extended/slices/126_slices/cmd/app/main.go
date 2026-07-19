package main

func main() {
	values := []int{5, 10, 9}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
