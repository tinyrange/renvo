package esp32s3

import "fmt"

// GPIO returns the pin capability used by the portable gpio and device packages.
func ExampleGPIO() {
	button := GPIO(41)
	fmt.Printf("%T\n", button)
	// Output: *esp32s3.Pin
}
