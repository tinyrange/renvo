package main

import "fmt"

func main() {
	if !(1 == 2) && !(2 == 3) && !((1) == 2) && ^1 == -2 && -("x")[0] == 136 && +"x"[0] == 120 {
		fmt.Print("PASS\n")
		return
	}
	fmt.Print("FAIL\n")
}
