//go:build m5tab5

package board

import (
	"renvo.dev/device/esp32p4"
	"renvo.dev/device/i2c"
)

type portAController struct {
	clock esp32p4.SystemTimer
	bus   i2c.BitBang
}

func (controller *portAController) Configure() error {
	if !InitPower() {
		return i2c.ErrBusy
	}
	return controller.bus.Configure()
}

func (controller *portAController) Tx(address uint16, write, read []byte) error {
	return controller.bus.Tx(address, write, read)
}

// PortA returns the Tab5 HY2.0-4P expansion connector (SDA GPIO53, SCL
// GPIO54). Construct it after the application has initialized its arena.
func PortA() i2c.Port {
	controller := &portAController{}
	controller.bus = i2c.NewBitBang(
		esp32p4.GPIO(53), esp32p4.GPIO(54), &controller.clock, 100000,
	)
	return i2c.DefinePort(controller, &controller.clock)
}
