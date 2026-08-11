// Package msc implements USB Mass Storage Bulk-Only Transport with the core
// transparent SCSI commands required by common operating systems.
package msc

import "renvo.dev/device/usb"

const blockSize = 512

// BlockDevice supplies 512-byte logical blocks.
type BlockDevice interface {
	BlockCount() uint32
	ReadBlock(block uint32, data []byte) error
	WriteBlock(block uint32, data []byte) error
}

type phase uint8

const (
	phaseCommand phase = iota
	phaseDataIn
	phaseDataOut
	phaseStatus
)

// Function is one SCSI transparent Bulk-Only Transport interface.
type Function struct {
	io                       usb.EndpointIO
	disk                     BlockDevice
	interfaceNumber, out, in uint8
	receive                  [64]byte
	block                    [blockSize]byte
	tx                       [64]byte
	tag                      uint32
	residue                  uint32
	status                   byte
	state                    phase
	blockIndex               uint32
	blocksRemaining          uint16
	blockOffset              int
	configured               bool
}

func New(disk BlockDevice) *Function { return &Function{disk: disk} }

func (f *Function) Bind(b *usb.DescriptorBuilder) error {
	f.interfaceNumber = b.Interface()
	var err error
	if f.out, err = b.Endpoint(usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if f.in, err = b.Endpoint(usb.In, usb.Bulk, 64, 0); err != nil {
		return err
	}
	if err = b.InterfaceDescriptor(f.interfaceNumber, 0, 2, 8, 6, 0x50, 0); err != nil {
		return err
	}
	if err = b.EndpointDescriptor(f.out, usb.Out, usb.Bulk, 64, 0); err != nil {
		return err
	}
	return b.EndpointDescriptor(f.in, usb.In, usb.Bulk, 64, 0)
}
func (f *Function) Attach(io usb.EndpointIO) { f.io = io }
func (f *Function) Control(setup usb.Setup, buffer []byte) int {
	if uint8(setup.Index) != f.interfaceNumber || setup.RequestType&0x60 != 0x20 {
		return usb.ControlNotHandled
	}
	if setup.Request == 0xfe && setup.RequestType&0x80 != 0 {
		buffer[0] = 0 // one logical unit
		return 1
	}
	if setup.Request == 0xff && setup.Length == 0 {
		f.resetTransport()
		return 0
	}
	return usb.ControlNotHandled
}
func (*Function) ControlOut(usb.Setup, []byte) bool { return false }
func (*Function) BOSDescriptor() []byte             { return nil }
func (f *Function) Configured(value bool) {
	f.configured = value
	if value {
		f.resetTransport()
	}
}
func (f *Function) resetTransport() {
	f.state, f.status, f.residue = phaseCommand, 0, 0
	if f.configured && f.io != nil {
		_ = f.io.EndpointReceive(f.out, f.receive[:])
	}
}

func le32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}
func be16(data []byte) uint16 { return uint16(data[0])<<8 | uint16(data[1]) }
func be32(data []byte) uint32 {
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

func (f *Function) command(data []byte) {
	if len(data) != 31 || le32(data) != 0x43425355 || data[14] == 0 || data[14] > 16 {
		f.io.StallEndpoint(f.out, usb.Out)
		return
	}
	f.tag, f.residue, f.status = le32(data[4:]), le32(data[8:]), 0
	command := data[15:31]
	switch command[0] {
	case 0x00: // TEST UNIT READY
		f.sendStatus()
	case 0x03: // REQUEST SENSE
		response := [18]byte{0x70, 0, 0, 0, 0, 0, 0, 10}
		f.sendData(response[:])
	case 0x12: // INQUIRY
		response := [36]byte{0, 0x80, 4, 2, 31}
		copy(response[8:16], "RENVO   ")
		copy(response[16:32], "USB BLOCK DEVICE")
		copy(response[32:36], "0001")
		f.sendData(response[:])
	case 0x1a: // MODE SENSE(6)
		f.sendData([]byte{3, 0, 0x80, 0})
	case 0x1e: // PREVENT/ALLOW MEDIUM REMOVAL
		f.sendStatus()
	case 0x25: // READ CAPACITY(10)
		last := f.disk.BlockCount() - 1
		response := [8]byte{byte(last >> 24), byte(last >> 16), byte(last >> 8), byte(last), 0, 0, 2, 0}
		f.sendData(response[:])
	case 0x28: // READ(10)
		f.blockIndex, f.blocksRemaining = be32(command[2:6]), be16(command[7:9])
		if f.blocksRemaining == 0 {
			f.sendStatus()
			return
		}
		if err := f.disk.ReadBlock(f.blockIndex, f.block[:]); err != nil {
			f.fail()
			return
		}
		f.blockOffset, f.state = 0, phaseDataIn
		f.sendNextBlockPacket()
	case 0x2a: // WRITE(10)
		f.blockIndex, f.blocksRemaining = be32(command[2:6]), be16(command[7:9])
		f.blockOffset, f.state = 0, phaseDataOut
		_ = f.io.EndpointReceive(f.out, f.receive[:])
	default:
		f.fail()
	}
}

func (f *Function) sendData(data []byte) {
	length := len(data)
	if length > int(f.residue) {
		length = int(f.residue)
	}
	f.residue -= uint32(length)
	copy(f.tx[:], data[:length])
	f.state = phaseStatus
	_ = f.io.EndpointSend(f.in, f.tx[:length])
}
func (f *Function) sendNextBlockPacket() {
	remaining := blockSize - f.blockOffset
	length := 64
	if remaining < length {
		length = remaining
	}
	if uint32(length) > f.residue {
		length = int(f.residue)
	}
	f.residue -= uint32(length)
	_ = f.io.EndpointSend(f.in, f.block[f.blockOffset:f.blockOffset+length])
	f.blockOffset += length
}
func (f *Function) sendStatus() {
	f.tx[0], f.tx[1], f.tx[2], f.tx[3] = 'U', 'S', 'B', 'S'
	f.tx[4], f.tx[5], f.tx[6], f.tx[7] = byte(f.tag), byte(f.tag>>8), byte(f.tag>>16), byte(f.tag>>24)
	f.tx[8], f.tx[9], f.tx[10], f.tx[11] = byte(f.residue), byte(f.residue>>8), byte(f.residue>>16), byte(f.residue>>24)
	f.tx[12] = f.status
	f.state = phaseCommand
	_ = f.io.EndpointSend(f.in, f.tx[:13])
}
func (f *Function) fail() { f.status = 1; f.sendStatus() }

func (f *Function) Handle(event usb.Event) {
	if event.Kind == usb.EventOut && event.Endpoint == f.out {
		if f.state == phaseCommand {
			f.command(event.Data)
			return
		}
		if f.state == phaseDataOut {
			count := copy(f.block[f.blockOffset:], event.Data)
			f.blockOffset += count
			f.residue -= uint32(count)
			if f.blockOffset == blockSize {
				if err := f.disk.WriteBlock(f.blockIndex, f.block[:]); err != nil {
					f.fail()
					return
				}
				f.blockIndex++
				f.blocksRemaining--
				f.blockOffset = 0
			}
			if f.blocksRemaining == 0 {
				f.sendStatus()
			} else {
				_ = f.io.EndpointReceive(f.out, f.receive[:])
			}
		}
	}
	if event.Kind == usb.EventInComplete && event.Endpoint == f.in {
		if f.state == phaseStatus {
			f.sendStatus()
			return
		}
		if f.state == phaseDataIn {
			if f.blockOffset == blockSize {
				f.blockIndex++
				f.blocksRemaining--
				f.blockOffset = 0
				if f.blocksRemaining != 0 {
					if err := f.disk.ReadBlock(f.blockIndex, f.block[:]); err != nil {
						f.fail()
						return
					}
				}
			}
			if f.blocksRemaining == 0 || f.residue == 0 {
				f.sendStatus()
			} else {
				f.sendNextBlockPacket()
			}
			return
		}
		if f.state == phaseCommand {
			_ = f.io.EndpointReceive(f.out, f.receive[:])
		}
	}
}
