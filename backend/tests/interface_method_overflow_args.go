package main

type overflowTransport interface {
	Tx(uint16, []byte, []byte) error
}

type overflowTransportImpl struct{}

func (*overflowTransportImpl) Tx(address uint16, write, read []byte) error {
	if address != 0x58 || len(write) != 2 || write[0] != 0x20 ||
		write[1] != 0x03 || read != nil {
		return overflowTransportError{}
	}
	return nil
}

type overflowTransportError struct{}

func (overflowTransportError) Error() string { return "bad arguments" }

func appMain() int {
	var transport overflowTransport = &overflowTransportImpl{}
	if err := transport.Tx(0x58, []byte{0x20, 0x03}, nil); err != nil {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
