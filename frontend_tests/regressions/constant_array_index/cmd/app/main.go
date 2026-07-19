package main

const length = 4

type values [length]int

func main() {
	var data values
	data[0] = 2
	data[1] = 5
	data[length-1] = 3
	index := 1
	if data[0]+data[index]+data[3] != 10 {
		return
	}
	print("PASS\n")
}
