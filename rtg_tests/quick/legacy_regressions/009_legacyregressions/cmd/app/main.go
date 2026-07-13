package main

const inc = 2

func main() {
	base++
	if base+inc+zero == 43 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
