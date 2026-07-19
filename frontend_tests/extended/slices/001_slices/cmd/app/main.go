package main

func main() {
	values := []int{1, 2, 3}
	values = append(values[1:2], 4)
	corpusOK := len(values) == 2 && values[0]+values[1] == 6
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
