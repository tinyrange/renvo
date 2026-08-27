package esp32p4

import "fmt"

// GPIO returns the pin capability used by the portable gpio and device packages.
func ExampleGPIO() {
	interrupt := GPIO(23)
	fmt.Printf("%T\n", interrupt)
	// Output: *esp32p4.Pin
}
