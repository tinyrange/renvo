package usb

// DescriptorBuilder allocates interface and endpoint numbers and writes one
// configuration descriptor into a caller-independent bounded buffer.
type DescriptorBuilder struct {
	data          [1024]byte
	length        int
	interfaces    uint8
	usedIn        [16]bool
	usedOut       [16]bool
	endpoints     [30]EndpointConfig
	endpointCount int
}

func newDescriptorBuilder() DescriptorBuilder {
	b := DescriptorBuilder{length: 9}
	b.data[0], b.data[1] = 9, 2
	b.data[5], b.data[6], b.data[7], b.data[8] = 1, 1, 0x80, 50
	return b
}

// Interface reserves and returns a contiguous interface number.
func (b *DescriptorBuilder) Interface() uint8 {
	result := b.interfaces
	b.interfaces++
	return result
}

// Endpoint reserves an endpoint number and records its hardware contract.
func (b *DescriptorBuilder) Endpoint(direction Direction, transfer TransferType, maxPacket uint16, interval uint8) (uint8, error) {
	if b.endpointCount == len(b.endpoints) {
		return 0, ErrEndpointOverflow
	}
	used, opposite := &b.usedOut, &b.usedIn
	if direction == In {
		used, opposite = &b.usedIn, &b.usedOut
	}
	number := uint8(0)
	// IN and OUT endpoints have independent addresses. Prefer pairing opposite
	// directions onto one endpoint number so small device controllers are not
	// needlessly exhausted by bidirectional functions.
	for candidate := uint8(1); candidate <= 15; candidate++ {
		if !used[candidate] && opposite[candidate] {
			number = candidate
			break
		}
	}
	if number == 0 {
		for candidate := uint8(1); candidate <= 15; candidate++ {
			if !used[candidate] {
				number = candidate
				break
			}
		}
	}
	if number == 0 {
		return 0, ErrEndpointOverflow
	}
	used[number] = true
	b.endpoints[b.endpointCount] = EndpointConfig{Number: number, Direction: direction, Transfer: transfer, MaxPacketSize: maxPacket, Interval: interval}
	b.endpointCount++
	return number, nil
}

// Append adds a complete descriptor fragment.
func (b *DescriptorBuilder) Append(data ...byte) error {
	if b.length+len(data) > len(b.data) {
		return ErrDescriptorOverflow
	}
	copy(b.data[b.length:], data)
	b.length += len(data)
	return nil
}

// InterfaceDescriptor appends a standard interface descriptor.
func (b *DescriptorBuilder) InterfaceDescriptor(number, alternate, endpoints, class, subclass, protocol, stringIndex uint8) error {
	return b.Append(9, 4, number, alternate, endpoints, class, subclass, protocol, stringIndex)
}

// EndpointDescriptor appends the standard descriptor for a reserved endpoint.
func (b *DescriptorBuilder) EndpointDescriptor(number uint8, direction Direction, transfer TransferType, maxPacket uint16, interval uint8) error {
	return b.EndpointDescriptorAttributes(number, direction, byte(transfer), maxPacket, interval)
}

// EndpointDescriptorAttributes appends an endpoint with the complete USB
// bmAttributes byte. Class packages use this for isochronous synchronization
// and usage bits that do not fit in TransferType alone.
func (b *DescriptorBuilder) EndpointDescriptorAttributes(number uint8, direction Direction, attributes uint8, maxPacket uint16, interval uint8) error {
	address := number
	if direction == In {
		address |= 0x80
	}
	return b.Append(7, 5, address, attributes, byte(maxPacket), byte(maxPacket>>8), interval)
}

func (b *DescriptorBuilder) finish() []byte {
	b.data[2], b.data[3] = byte(b.length), byte(b.length>>8)
	b.data[4] = b.interfaces
	return b.data[:b.length]
}

func deviceDescriptor(config Config) [18]byte {
	class, subclass, protocol := byte(0), byte(0), byte(0)
	if len(config.Functions) > 1 {
		class, subclass, protocol = 0xef, 2, 1
	}
	return [18]byte{
		18, 1, 0x00, 0x02, class, subclass, protocol, 64,
		byte(config.VendorID), byte(config.VendorID >> 8),
		byte(config.ProductID), byte(config.ProductID >> 8),
		byte(config.DeviceBCD), byte(config.DeviceBCD >> 8),
		1, 2, 3, 1,
	}
}

func stringDescriptor(value string, output []byte) int {
	if len(output) < 2 {
		return 0
	}
	limit := len(value)
	if limit > (len(output)-2)/2 {
		limit = (len(output) - 2) / 2
	}
	output[0], output[1] = byte(2+limit*2), 3
	for index := 0; index < limit; index++ {
		output[2+index*2] = value[index]
		output[3+index*2] = 0
	}
	return 2 + limit*2
}
