package fat32

import "fmt"

func ExampleClean() {
	fmt.Println(Clean("/logs", "../settings.txt"))
	// Output: /settings.txt
}
