package button

import "fmt"

func ExampleDebouncer_Update() {
	key := Debouncer{DelayMS: 20}
	fmt.Println(key.Update(true, 100))
	fmt.Println(key.Update(true, 120), key.Pressed)
	// Output:
	// false
	// true true
}
