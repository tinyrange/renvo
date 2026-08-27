package st7121_test

import "renvo.dev/device/display/st7121"

func ExampleInitialize() {
	var transport st7121.Transport // Supplied by the board's DSI driver.
	st7121.Initialize(transport)
}
