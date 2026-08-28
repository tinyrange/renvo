//go:build m5nanoc6

package i2c_test

import (
	"errors"

	"renvo.dev/device/board"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/i2c"
)

// Device keeps a peripheral's seven-bit address separate from register addresses.
func ExampleBus_Device() {
	bus := i2c.New(board.Grove)
	accelerometer := bus.Device(0x53)

	data := make([]byte, 2)
	_, _ = accelerometer.ReadAt(data, 0x32)
}

// Tx is useful when a device uses a multi-byte register address or a custom
// command framing that ReadAt and WriteAt do not represent.
func ExampleBus_Tx() {
	bus := i2c.New(board.Grove)
	register := []byte{0x00, 0x0f} // A 16-bit register address.
	response := make([]byte, 4)
	_ = bus.Tx(0x50, register, response)
}

func ExampleDevice_ReadAt() {
	device := i2c.New(board.Grove).Device(0x53)
	data := make([]byte, 6)
	if count, err := device.ReadAt(data, 0x32); err == nil {
		data = data[:count]
	}
}

func ExampleDevice_WriteAt() {
	device := i2c.New(board.Grove).Device(0x53)
	_, _ = device.WriteAt([]byte{0x08}, 0x2d) // Enable measurement mode.
}

// Read is for devices that stream data without a register-selection write.
func ExampleDevice_Read() {
	device := i2c.New(board.Grove).Device(0x48)
	measurement := make([]byte, 2)
	_, _ = device.Read(measurement)
}

// Write is for command-oriented devices that do not expose byte registers.
func ExampleDevice_Write() {
	device := i2c.New(board.Grove).Device(0x58)
	_, _ = device.Write([]byte{0x20, 0x03}) // Start air-quality measurement.
}

// NewBitBang lets a board package define I2C on any two open-drain-capable pins.
func ExampleNewBitBang() {
	controller := i2c.NewBitBang(
		esp32c6.GPIO(2), // SDA
		esp32c6.GPIO(1), // SCL
		&board.Clock,
		100_000,
	)
	port := i2c.DefinePort(&controller, &board.Clock)
	bus := i2c.New(port)
	_ = bus.Tx(0x50, []byte{0}, nil)
}

// DefinePort is used by board packages to publish a connector without exposing
// the controller's concrete type to applications and drivers.
func ExampleDefinePort() {
	controller := i2c.NewBitBang(esp32c6.GPIO(2), esp32c6.GPIO(1), &board.Clock, 100_000)
	grove := i2c.DefinePort(&controller, &board.Clock)
	bus := i2c.New(grove)
	_ = bus.Tx(0x50, []byte{0x00}, nil)
}

// Tx implements the Controller interface for board-defined software I2C ports.
func ExampleBitBang_Tx() {
	controller := i2c.NewBitBang(esp32c6.GPIO(2), esp32c6.GPIO(1), &board.Clock, 100_000)
	if err := controller.Configure(); err != nil {
		return
	}
	response := make([]byte, 2)
	_ = controller.Tx(0x48, []byte{0x00}, response)
}

func ExampleBus_DelayMilliseconds() {
	bus := i2c.New(board.Grove)
	device := bus.Device(0x58)
	_, _ = device.Write([]byte{0x20, 0x03})
	bus.DelayMilliseconds(15) // Wait for the command to complete.
}

func ExampleOperationError_Error() {
	device := i2c.New(board.Grove).Device(0x53)
	_, err := device.ReadAt(make([]byte, 2), 0x32)
	if operation, ok := err.(*i2c.OperationError); ok {
		message := operation.Error()
		_, _ = message, operation.Address
	}
}

func ExampleOperationError_Unwrap() {
	device := i2c.New(board.Grove).Device(0x53)
	_, err := device.ReadAt(make([]byte, 2), 0x32)
	if errors.Is(err, i2c.ErrNAK) {
		// The peripheral is absent or did not acknowledge this transaction.
	}
}
