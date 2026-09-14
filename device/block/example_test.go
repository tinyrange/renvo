package block

import "fmt"

func ExampleValid() {
	fmt.Println(Valid(1024, 1023, Size))
	fmt.Println(Valid(1024, 1023, 2*Size))
	// Output:
	// true
	// false
}
