package esp32s3

import (
	"errors"

	"renvo.dev/device/mmio"
	"renvo.dev/device/usb"
)

const (
	usbCoreBase = uintptr(0x60080000)
	usbWrapBase = uintptr(0x60039000)
	rtcUsbConf  = uintptr(0x60008120)

	gahbCfg   = uintptr(0x008)
	gusbCfg   = uintptr(0x00c)
	grstCtl   = uintptr(0x010)
	gintSts   = uintptr(0x014)
	gintMsk   = uintptr(0x018)
	grxStsP   = uintptr(0x020)
	grxFsiz   = uintptr(0x024)
	gnptxFsiz = uintptr(0x028)
	gdfifoCfg = uintptr(0x05c)
	dcfg      = uintptr(0x800)
	dctl      = uintptr(0x804)
	diepMsk   = uintptr(0x810)
	doepMsk   = uintptr(0x814)
	daint     = uintptr(0x818)
	daintMsk  = uintptr(0x81c)
	gotgCtl   = uintptr(0x000)

	gintRxFIFO   = uint32(1 << 4)
	gintReset    = uint32(1 << 12)
	gintEnumDone = uint32(1 << 13)
	gintIn       = uint32(1 << 18)
	gintOut      = uint32(1 << 19)

	epEnable   = uint32(1 << 31)
	epDisable  = uint32(1 << 30)
	epClearNAK = uint32(1 << 26)
	epSetNAK   = uint32(1 << 27)
	epActive   = uint32(1 << 15)
	epStall    = uint32(1 << 21)

	wrapClockEnable      = uint32(1 << 31)
	wrapFIFOForceUp      = uint32(1 << 22)
	wrapPHYClockForceOn  = uint32(1 << 20)
	wrapAHBClockForceOn  = uint32(1 << 19)
	wrapUSBPadEnable     = uint32(1 << 18)
	wrapDMPulldown       = uint32(1 << 16)
	wrapDMPullup         = uint32(1 << 15)
	wrapDPPulldown       = uint32(1 << 14)
	wrapDPPullup         = uint32(1 << 13)
	wrapPadPullOverride  = uint32(1 << 12)
	wrapFIFOForceDown    = uint32(1 << 3)
	wrapExternalPHY      = uint32(1 << 2)
	rtcSoftwarePHYSelect = uint32(1 << 20)
	rtcSelectUSBOTG      = uint32(1 << 19)
)

var errDWC2Endpoint = errors.New("esp32s3 dwc2 endpoint is unsupported")

// DWC2 is the ESP32-S3 full-speed USB OTG peripheral in device mode. It uses
// the core's slave FIFO interface so callers do not need DMA-capable buffers.
type DWC2 struct {
	receive       [7][]byte
	configuredIn  [7]bool
	configuredOut [7]bool
	maxPacketIn   [7]uint16
	maxPacketOut  [7]uint16
	fifoTop       uint16
	rxWords       uint16
	started       bool
}

func coreRegister(offset uintptr) mmio.Register32 { return mmio.Register32(usbCoreBase + offset) }
func inControl(endpoint uint8) mmio.Register32 {
	return coreRegister(0x900 + uintptr(endpoint)*0x20)
}
func inInterrupt(endpoint uint8) mmio.Register32 {
	return coreRegister(0x908 + uintptr(endpoint)*0x20)
}
func inSize(endpoint uint8) mmio.Register32 {
	return coreRegister(0x910 + uintptr(endpoint)*0x20)
}
func outControl(endpoint uint8) mmio.Register32 {
	return coreRegister(0xb00 + uintptr(endpoint)*0x20)
}
func outInterrupt(endpoint uint8) mmio.Register32 {
	return coreRegister(0xb08 + uintptr(endpoint)*0x20)
}
func outSize(endpoint uint8) mmio.Register32 {
	return coreRegister(0xb10 + uintptr(endpoint)*0x20)
}
func fifo(endpoint uint8) mmio.Register32 {
	return coreRegister(0x1000 + uintptr(endpoint)*0x1000)
}

func endpointType(transfer usb.TransferType) uint32 {
	return uint32(transfer) << 18
}

func (d *DWC2) waitAHBIdle() {
	for coreRegister(grstCtl).Load()&(1<<31) == 0 {
	}
}

func (d *DWC2) resetCore() {
	d.waitAHBIdle()
	coreRegister(grstCtl).Store(1)
	for coreRegister(grstCtl).Load()&1 != 0 {
	}
	d.waitAHBIdle()
}

// Start powers and initializes the native PHY and the DWC2 device core.
func (d *DWC2) Start() error {
	if d.started {
		return nil
	}
	// ESP32-S3's internal PHY is shared with the fixed USB Serial/JTAG block.
	// Override the eFuse selection and route it to DWC2 before touching the
	// core. Hold both data lines down until Connect exposes the D+ pull-up.
	mmio.Register32(rtcUsbConf).Replace(0, rtcSoftwarePHYSelect|rtcSelectUSBOTG)
	mmio.Register32(usbWrapBase).Replace(
		wrapFIFOForceDown|wrapExternalPHY|wrapDPPullup|wrapDMPullup,
		wrapClockEnable|wrapFIFOForceUp|wrapPHYClockForceOn|wrapAHBClockForceOn|
			wrapUSBPadEnable|wrapPadPullOverride|wrapDPPulldown|wrapDMPulldown,
	)
	coreRegister(gusbCfg).Store(coreRegister(gusbCfg).Load() | 1<<30)
	d.resetCore()
	coreRegister(gusbCfg).Store(coreRegister(gusbCfg).Load() | 1<<30)
	coreRegister(dctl).Store(coreRegister(dctl).Load() | 1<<1)
	// This board does not wire a separate VBUS-sense input. Force the DWC2
	// B-session valid state and select its full-speed PHY encoding.
	coreRegister(gotgCtl).Replace(1<<4, 1<<6|1<<7)

	// The S3 has 256 words shared by every RX and TX FIFO. Reserve a
	// specification-sized RX area for EP0 and allocate IN FIFOs downwards as
	// endpoints are opened. Keeping free space between them lets larger OUT
	// endpoints grow the shared RX FIFO safely.
	d.fifoTop = 256
	d.rxWords = receiveFIFOWords(64)
	coreRegister(gdfifoCfg).Store(256 | 256<<16)
	coreRegister(grxFsiz).Store(uint32(d.rxWords))
	d.fifoTop -= 16
	coreRegister(gnptxFsiz).Store(uint32(d.fifoTop) | 16<<16)
	coreRegister(dcfg).Replace(3|(0x7f<<4), 3|1<<2)
	coreRegister(diepMsk).Store(1)
	coreRegister(doepMsk).Store(1 | 1<<3)
	coreRegister(daintMsk).Store(1 | 1<<16)
	coreRegister(gintSts).Store(0xffffffff)
	coreRegister(gintMsk).Store(gintRxFIFO | gintReset | gintEnumDone | gintIn | gintOut)
	coreRegister(gahbCfg).Store(coreRegister(gahbCfg).Load() | 1)
	d.configureControlEndpoint()
	d.started = true
	return nil
}

func (d *DWC2) configureControlEndpoint() {
	d.configuredIn[0], d.configuredOut[0] = true, true
	d.maxPacketIn[0], d.maxPacketOut[0] = 64, 64
	inControl(0).Store(epActive)
	outControl(0).Store(epActive)
	d.armSetup()
}

func (d *DWC2) armSetup() {
	outSize(0).Store(64 | 1<<19 | 3<<29)
	outControl(0).Store(outControl(0).Load() | epEnable | epClearNAK)
}

func (d *DWC2) Connect() error {
	// Return pull control to DWC2. Its soft-disconnect state then owns the
	// normal full-speed D+ pull-up while the wrapper remains responsible only
	// for forcing a visible disconnect.
	mmio.Register32(usbWrapBase).Replace(
		wrapPadPullOverride|wrapDPPullup|wrapDPPulldown|wrapDMPullup|wrapDMPulldown,
		0,
	)
	coreRegister(dctl).Store(coreRegister(dctl).Load() &^ (1 << 1))
	return nil
}

func (d *DWC2) Disconnect() {
	coreRegister(dctl).Store(coreRegister(dctl).Load() | 1<<1)
	mmio.Register32(usbWrapBase).Replace(wrapDPPullup|wrapDMPullup, wrapDPPulldown|wrapDMPulldown)
}

func (d *DWC2) OpenEndpoint(config usb.EndpointConfig) error {
	if config.Number == 0 || config.Number > 6 || config.MaxPacketSize == 0 || config.MaxPacketSize > 1023 {
		return errDWC2Endpoint
	}
	control := epActive | endpointType(config.Transfer) | uint32(config.MaxPacketSize)
	if config.Direction == usb.In {
		words := uint16((config.MaxPacketSize + 3) / 4)
		if words == 0 || words > d.fifoTop || d.fifoTop-words < d.rxWords {
			return errDWC2Endpoint
		}
		d.fifoTop -= words
		coreRegister(0x100 + uintptr(config.Number-1)*4).Store(uint32(d.fifoTop) | uint32(words)<<16)
		control |= uint32(config.Number) << 22
		inControl(config.Number).Store(control)
		d.configuredIn[config.Number] = true
		d.maxPacketIn[config.Number] = config.MaxPacketSize
	} else {
		required := receiveFIFOWords(config.MaxPacketSize)
		if required > d.fifoTop {
			return errDWC2Endpoint
		}
		if required > d.rxWords {
			d.rxWords = required
			coreRegister(grxFsiz).Store(uint32(d.rxWords))
		}
		outControl(config.Number).Store(control)
		d.configuredOut[config.Number] = true
		d.maxPacketOut[config.Number] = config.MaxPacketSize
	}
	mask := coreRegister(daintMsk).Load()
	if config.Direction == usb.In {
		mask |= 1 << config.Number
	} else {
		mask |= 1 << (16 + config.Number)
	}
	coreRegister(daintMsk).Store(mask)
	return nil
}

// receiveFIFOWords follows the DWC2 device-mode sizing formula for one
// setup queue, two largest packets, and bookkeeping for seven OUT endpoints.
func receiveFIFOWords(maxPacket uint16) uint16 {
	packetWords := (maxPacket + 3) / 4
	return 13 + 1 + 2*(packetWords+1) + 2*7
}

func (d *DWC2) Send(endpoint uint8, data []byte) error {
	if endpoint > 6 || !d.configuredIn[endpoint] || len(data) > 1023 {
		return errDWC2Endpoint
	}
	packets := uint32(1)
	if len(data) > 0 {
		maxPacket := int(d.maxPacketIn[endpoint])
		packets = uint32((len(data) + maxPacket - 1) / maxPacket)
	}
	inSize(endpoint).Store(uint32(len(data)) | packets<<19)
	inControl(endpoint).Store(inControl(endpoint).Load() | epEnable | epClearNAK)
	for offset := 0; offset < len(data); offset += 4 {
		var word uint32
		for index := 0; index < 4 && offset+index < len(data); index++ {
			word |= uint32(data[offset+index]) << uint(index*8)
		}
		fifo(endpoint).Store(word)
	}
	return nil
}

func (d *DWC2) Receive(endpoint uint8, buffer []byte) error {
	if endpoint > 6 || !d.configuredOut[endpoint] || len(buffer) > 1023 {
		return errDWC2Endpoint
	}
	d.receive[endpoint] = buffer
	packets := uint32(1)
	if len(buffer) > 0 {
		maxPacket := int(d.maxPacketOut[endpoint])
		packets = uint32((len(buffer) + maxPacket - 1) / maxPacket)
	}
	outSize(endpoint).Store(uint32(len(buffer)) | packets<<19)
	outControl(endpoint).Store(outControl(endpoint).Load() | epEnable | epClearNAK)
	return nil
}

func (d *DWC2) Stall(endpoint uint8, direction usb.Direction) {
	if endpoint > 6 {
		return
	}
	if direction == usb.In {
		inControl(endpoint).Store(inControl(endpoint).Load() | epStall | epSetNAK)
	} else {
		outControl(endpoint).Store(outControl(endpoint).Load() | epStall | epSetNAK)
	}
}

func (d *DWC2) SetAddress(address uint8) {
	coreRegister(dcfg).Replace(0x7f<<4, uint32(address&0x7f)<<4)
}

func readFIFO(data []byte) {
	for offset := 0; offset < len(data); offset += 4 {
		word := fifo(0).Load()
		for index := 0; index < 4 && offset+index < len(data); index++ {
			data[offset+index] = byte(word >> uint(index*8))
		}
	}
}

// Poll returns at most one controller event and leaves remaining causes for the
// next bounded device poll iteration.
func (d *DWC2) Poll(event *usb.Event) bool {
	status := coreRegister(gintSts).Load() & coreRegister(gintMsk).Load()
	if status&gintReset != 0 {
		coreRegister(gintSts).Store(gintReset | gintEnumDone)
		d.configureControlEndpoint()
		event.Kind = usb.EventReset
		return true
	}
	if status&gintEnumDone != 0 {
		coreRegister(gintSts).Store(gintEnumDone)
		d.configureControlEndpoint()
		return false
	}
	if status&gintRxFIFO != 0 {
		receiveStatus := coreRegister(grxStsP).Load()
		endpoint := uint8(receiveStatus & 15)
		length := int((receiveStatus >> 4) & 0x7ff)
		if endpoint <= 6 && receiveStatus&(15<<17) == 6<<17 {
			readFIFO(event.Setup[:])
			event.Kind = usb.EventSetup
			event.Endpoint = 0
			return true
		}
		if endpoint <= 6 && receiveStatus&(15<<17) == 2<<17 {
			buffer := d.receive[endpoint]
			if length > len(buffer) {
				length = len(buffer)
			}
			readFIFO(buffer[:length])
			event.Kind = usb.EventOut
			event.Endpoint = endpoint
			event.Data = buffer[:length]
			return true
		}
	}
	interrupts := coreRegister(daint).Load() & coreRegister(daintMsk).Load()
	for endpoint := uint8(0); endpoint <= 6; endpoint++ {
		if interrupts&(1<<endpoint) != 0 {
			cause := inInterrupt(endpoint).Load()
			inInterrupt(endpoint).Store(cause)
			if cause&1 != 0 {
				event.Kind = usb.EventInComplete
				event.Endpoint = endpoint
				return true
			}
		}
		if interrupts&(1<<(16+endpoint)) != 0 {
			cause := outInterrupt(endpoint).Load()
			outInterrupt(endpoint).Store(cause)
			if endpoint == 0 && cause&(1<<3) != 0 {
				d.armSetup()
			}
		}
	}
	return false
}
