package main

import "fmt"

func main() {
	err := fmt.Errorf("%s %d", "code", 7)
	if err.Error() == "code 7" {
		print("PASS\n")
	}
}
