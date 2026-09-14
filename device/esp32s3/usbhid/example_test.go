package usbhid

// Poll must be called frequently while the USB controller is active.
func ExamplePoll() {
	// After Start with Keyboard enabled, bit 0 holds A and bit 1 holds B.
	Poll(1)
	Poll(3)
	Poll(0) // Release both keys.
}
