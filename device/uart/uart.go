// Package uart provides board-independent asynchronous serial transmitters.
package uart

type uartError string

func (e uartError) Error() string { return string(e) }

const (
	// ErrInvalidBaud reports a rate the selected controller cannot generate.
	ErrInvalidBaud uartError = "unsupported UART baud rate"
	// ErrNotConfigured reports direct use of a controller before Configure.
	ErrNotConfigured uartError = "UART transmitter is not configured"
)

// Controller is the capability supplied by hardware or software UART engines.
type Controller interface {
	Configure(baud uint32) error
	Write([]byte) (int, error)
}

// Port is a board-owned transmit-capable serial connector. Applications open
// it at a chosen baud rate rather than depending on chip pin numbers.
type Port struct {
	controller Controller
}

// DefinePort constructs a board port around controller.
func DefinePort(controller Controller) Port { return Port{controller: controller} }

// TX is a lazily configured serial transmitter.
type TX struct {
	controller  Controller
	baud        uint32
	initialized bool
	err         error
}

// New opens port as an 8-data-bit, no-parity, one-stop-bit transmitter.
// Hardware is not touched until the first Write.
func New(port Port, baud uint32) *TX {
	return &TX{controller: port.controller, baud: baud}
}

// Write transmits data and implements io.Writer.
func (t *TX) Write(data []byte) (int, error) {
	if !t.initialized {
		t.err = t.controller.Configure(t.baud)
		t.initialized = true
	}
	if t.err != nil {
		return 0, t.err
	}
	return t.controller.Write(data)
}
