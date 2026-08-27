package main

/*
int cAdd(int left, int right);
int cCallGo(int value);
*/
import "C"

//export goDouble
func goDouble(value int) int { return value * 2 }

func main() {
	if C.cAdd(20, 22) == 42 && C.cCallGo(21) == 42 {
		print("PASS\n")
	}
}
