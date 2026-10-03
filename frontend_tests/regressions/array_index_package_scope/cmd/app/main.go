package main

func main() {
	const size = 99
	data[0] = 7
	data[3] = 9
	if len(data) != 4 || data[0] != 7 || data[3] != 9 || size != 99 {
		panic("array declaration scope")
	}
	println("PASS")
}
