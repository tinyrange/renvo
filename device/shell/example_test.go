package shell

import "fmt"

func ExampleSplit() {
	args, err := Split(`cat "daily notes.txt"`)
	fmt.Printf("%q %v\n", args, err)
	// Output: ["cat" "daily notes.txt"] <nil>
}
