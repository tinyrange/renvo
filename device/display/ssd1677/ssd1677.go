// Package ssd1677 implements the command protocol for an SSD1677 e-paper
// controller. Board packages own pin wiring, SPI setup, reset, and power.
package ssd1677

const (
	// Width and Height are the SSD1677 RAM dimensions used by the PaperMono-Lite.
	// The controller axes are rotated relative to the 480x800 visible panel.
	Width       = 800
	Height      = 480
	BytesPerRow = Width / 8
	FrameSize   = BytesPerRow * Height

	defaultBusyTimeoutMilliseconds = uint32(15000)
	partialRefreshLimit            = uint8(10)
)

const (
	commandSoftReset        = byte(0x12)
	commandDeepSleep        = byte(0x10)
	commandMasterActivation = byte(0x20)
	commandUpdateControl1   = byte(0x21)
	commandUpdateControl    = byte(0x22)
	commandWriteLUT         = byte(0x32)
	commandGateVoltage      = byte(0x03)
	commandSourceVoltage    = byte(0x04)
	commandWriteVCOM        = byte(0x2c)
	commandWriteRAM1        = byte(0x24)
	commandWriteRAM2        = byte(0x26)
	commandDataEntryMode    = byte(0x11)
	commandRAMXRange        = byte(0x44)
	commandRAMYRange        = byte(0x45)
	commandRAMXCounter      = byte(0x4e)
	commandRAMYCounter      = byte(0x4f)
)

// fastDifferentialLUT is M5Stack/M5GFX's two-clause BSD (FreeBSD variant)
// short Mode 2 monochrome waveform for the PaperMono SSD1677 panel:
// https://github.com/m5stack/M5GFX/blob/master/src/lgfx/v1/panel/Panel_SSD1677.cpp
// RAM 1 holds the next image and RAM 2 the previous image, selecting hold,
// black-to-white, and white-to-black transition groups.
var fastDifferentialLUT = [110]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x6a, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x95, 0x50, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x02, 0x02, 0x02, 0x02, 0x00,
	0x01, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
	0x8f, 0x8f, 0x8f, 0x8f, 0x8f,
	0x17, 0x46, 0xa8, 0x32, 0x30,
}

// Transport is the board-owned synchronous SSD1677 bus. Begin selects the
// controller and sends one command byte; Data may be called repeatedly before
// End releases chip select. Busy is true while the controller is processing.
type Transport interface {
	Reset() error
	Begin(command byte) error
	Data(data []byte) error
	End() error
	Busy() bool
	Milliseconds() uint32
	DelayMilliseconds(milliseconds uint32)
}

// GrayPlaneSource fills one packed 100-byte row at a time. It permits full
// four-gray refresh without keeping two complete 48,000-byte planes resident.
// Plane is zero for RAM 1 and one for RAM 2; row is in panel-native order.
type GrayPlaneSource interface {
	FillGrayRow(plane, row int, destination []byte) error
}

// RefreshPoller performs bounded application work while an e-paper waveform is
// active. Implementations must not mutate the framebuffer being presented.
type RefreshPoller interface {
	PollDuringRefresh() error
}

// Monochrome is one packed 1-bit panel-native framebuffer. Bits are MSB first;
// one means white and zero means black. The transfer path reverses the SSD1677
// gate counter so row zero remains the visible top of the native surface.
type Monochrome [FrameSize]byte

// Fill sets every pixel to white or black.
func (frame *Monochrome) Fill(white bool) {
	value := byte(0)
	if white {
		value = 0xff
	}
	for index := range frame {
		frame[index] = value
	}
}

// Set changes one controller-native pixel. It returns false outside the
// 800x480 RAM area.
func (frame *Monochrome) Set(x, y int, white bool) bool {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return false
	}
	index := y*BytesPerRow + x/8
	mask := byte(0x80 >> uint(x&7))
	if white {
		frame[index] |= mask
	} else {
		frame[index] &^= mask
	}
	return true
}

// Gray is one of the four levels supported by the controller OTP waveform.
type Gray byte

const (
	White     Gray = 0
	LightGray Gray = 1
	DarkGray  Gray = 2
	Black     Gray = 3
)

// PackGrayRow encodes 800 four-level pixels into two 100-byte packed planes.
// The OTP encoding is white=00, light gray=10, dark gray=01, black=11.
func PackGrayRow(pixels []Gray, plane1, plane2 []byte) error {
	if len(pixels) != Width || len(plane1) != BytesPerRow || len(plane2) != BytesPerRow {
		return ErrFrameSize
	}
	for index := range plane1 {
		plane1[index] = 0
		plane2[index] = 0
	}
	for x, pixel := range pixels {
		if pixel > Black {
			return ErrGrayLevel
		}
		mask := byte(0x80 >> uint(x&7))
		if pixel == LightGray || pixel == Black {
			plane1[x/8] |= mask
		}
		if pixel == DarkGray || pixel == Black {
			plane2[x/8] |= mask
		}
	}
	return nil
}

// Device tracks the monochrome baseline required by partial refreshes.
type Device struct {
	transport      Transport
	busyTimeout    uint32
	baseline       bool
	partialRefresh uint8
	fastActive     bool
}

// New returns a protocol driver with a 15-second BUSY timeout, matching the
// official M5Stack OTP demo's bounded wait.
func New(transport Transport) *Device {
	return &Device{transport: transport, busyTimeout: defaultBusyTimeoutMilliseconds}
}

// SetBusyTimeout overrides the BUSY timeout. A zero duration is rejected.
func (device *Device) SetBusyTimeout(milliseconds uint32) error {
	if milliseconds == 0 {
		return ErrTimeout
	}
	device.busyTimeout = milliseconds
	return nil
}

// HasBaseline reports whether a successful full monochrome refresh established
// the controller RAM state required by PartialMonochrome.
func (device *Device) HasBaseline() bool { return device.baseline }

// PartialRefreshes reports the number of differential updates since the last
// successful full monochrome refresh.
func (device *Device) PartialRefreshes() int { return int(device.partialRefresh) }

// InvalidateBaseline must be called after panel power is removed. SSD1677 RAM
// cannot be assumed to survive a board-level power cycle.
func (device *Device) InvalidateBaseline() {
	device.baseline = false
	device.partialRefresh = 0
	device.fastActive = false
}

// FullMonochrome writes a packed frame and applies the controller's OTP Mode 1
// waveform. Both RAM planes receive the final image, establishing a baseline.
func (device *Device) FullMonochrome(frame []byte) error {
	return device.fullMonochrome(frame, nil)
}

func (device *Device) fullMonochrome(frame []byte, poller RefreshPoller) error {
	if len(frame) != FrameSize {
		return ErrFrameSize
	}
	device.baseline = false
	device.partialRefresh = 0
	device.fastActive = false
	if err := device.resetAndInitializeMonochrome(); err != nil {
		return err
	}
	if err := device.command(commandUpdateControl, []byte{0xf8}); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM1, frame, true); err != nil {
		return err
	}
	if err := device.command(commandMasterActivation, nil); err != nil {
		return err
	}
	if err := device.waitReadyWithPoller(poller); err != nil {
		return err
	}
	if err := device.command(commandUpdateControl, []byte{0x14}); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM2, frame, false); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM1, frame, false); err != nil {
		return err
	}
	if err := device.command(commandMasterActivation, nil); err != nil {
		return err
	}
	if err := device.waitReadyWithPoller(poller); err != nil {
		return err
	}
	if err := device.deepSleep(); err != nil {
		return err
	}
	device.baseline = true
	return nil
}

// PartialMonochrome applies the built-in OTP partial waveform to a complete
// next frame. It requires an explicit full-refresh baseline. After ten partial
// updates, the next request automatically becomes a full refresh so
// differential updates cannot build up indefinitely.
func (device *Device) PartialMonochrome(frame []byte) error {
	if len(frame) != FrameSize {
		return ErrFrameSize
	}
	if !device.baseline {
		return ErrNoBaseline
	}
	if device.partialRefresh >= partialRefreshLimit {
		return device.FullMonochrome(frame)
	}
	device.fastActive = false
	if err := device.transport.Reset(); err != nil {
		return err
	}
	if err := device.waitReady(); err != nil {
		return err
	}
	if err := device.command(0x3c, []byte{0x80}); err != nil {
		return err
	}
	if err := device.setRAMWindow(0, 0, Width, Height); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM1, frame, false); err != nil {
		return err
	}
	if err := device.command(0x21, []byte{0x00}); err != nil {
		return err
	}
	if err := device.command(commandUpdateControl, []byte{0xff}); err != nil {
		return err
	}
	if err := device.command(commandMasterActivation, nil); err != nil {
		return err
	}
	if err := device.waitReady(); err != nil {
		device.baseline = false
		return err
	}
	if err := device.deepSleep(); err != nil {
		device.baseline = false
		return err
	}
	device.partialRefresh++
	return nil
}

// FastMonochrome applies M5Stack's short differential waveform without putting
// the controller back into deep sleep. Both complete transition planes are
// transferred because the SSD1677 advances its internal Mode 2 RAM face on
// every activation. RAM 2 is synchronized to frame for the next request. A
// successful full refresh baseline is required; the eleventh differential
// request is promoted to a recovery full refresh.
func (device *Device) FastMonochrome(frame []byte, poller RefreshPoller) error {
	if len(frame) != FrameSize {
		return ErrFrameSize
	}
	if !device.baseline {
		return ErrNoBaseline
	}
	if device.partialRefresh >= partialRefreshLimit {
		return device.fullMonochrome(frame, poller)
	}
	wasActive := device.fastActive
	if !wasActive {
		// Hardware reset exits deep sleep while retaining the two RAM planes
		// established by FullMonochrome. A software reset here would discard the
		// differential baseline; this is the wake sequence used by M5Stack's OTP
		// partial-refresh example.
		if err := device.transport.Reset(); err != nil {
			return device.invalidateAfterFastError(err)
		}
		if err := device.waitReady(); err != nil {
			return device.invalidateAfterFastError(err)
		}
		if err := device.command(0x3c, []byte{0x80}); err != nil {
			return device.invalidateAfterFastError(err)
		}
	} else if err := device.waitReady(); err != nil {
		return device.invalidateAfterFastError(err)
	}
	if err := device.writeRAM(commandWriteRAM1, frame, false); err != nil {
		return device.invalidateAfterFastError(err)
	}
	if err := device.writeFastDifferentialLUT(); err != nil {
		return device.invalidateAfterFastError(err)
	}
	if err := device.command(commandUpdateControl1, []byte{0x00}); err != nil {
		return device.invalidateAfterFastError(err)
	}
	update := byte(0x0c)
	if !wasActive {
		update |= 0xc0
	}
	if err := device.command(commandUpdateControl, []byte{update}); err != nil {
		return device.invalidateAfterFastError(err)
	}
	if err := device.command(commandMasterActivation, nil); err != nil {
		return device.invalidateAfterFastError(err)
	}
	if err := device.waitReadyWithPoller(poller); err != nil {
		return device.invalidateAfterFastError(err)
	}
	// Preserve the new optical state as the previous plane for the next live
	// update without retaining a second 48 KiB application framebuffer.
	if err := device.writeRAM(commandWriteRAM2, frame, false); err != nil {
		return device.invalidateAfterFastError(err)
	}
	device.fastActive = true
	device.partialRefresh++
	return nil
}

// FullGray writes two packed bit planes and applies the built-in four-gray OTP
// waveform. The gray result cannot serve as a differential monochrome baseline.
func (device *Device) FullGray(plane1, plane2 []byte) error {
	if len(plane1) != FrameSize || len(plane2) != FrameSize {
		return ErrFrameSize
	}
	if err := device.initializeGray(); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM1, plane1, false); err != nil {
		return err
	}
	if err := device.writeRAM(commandWriteRAM2, plane2, false); err != nil {
		return err
	}
	return device.finishGray()
}

// FullGrayStream performs the same two-plane OTP refresh while requesting one
// already-packed row at a time. This is the preferred path on internal-RAM-only
// targets which cannot safely retain both complete planes.
func (device *Device) FullGrayStream(source GrayPlaneSource) error {
	if err := device.initializeGray(); err != nil {
		return err
	}
	if err := device.writeGrayPlane(commandWriteRAM1, source, 0); err != nil {
		return err
	}
	if err := device.writeGrayPlane(commandWriteRAM2, source, 1); err != nil {
		return err
	}
	return device.finishGray()
}

func (device *Device) initializeGray() error {
	device.baseline = false
	device.partialRefresh = 0
	device.fastActive = false
	if err := device.transport.Reset(); err != nil {
		return err
	}
	if err := device.softwareReset(); err != nil {
		return err
	}
	if err := device.command(0x0c, []byte{0xae, 0xc7, 0xc3, 0xc0, 0x80}); err != nil {
		return err
	}
	if err := device.command(0x01, []byte{0xdf, 0x01, 0x02}); err != nil {
		return err
	}
	if err := device.setRAMWindow(0, 0, Width, Height); err != nil {
		return err
	}
	if err := device.command(0x3c, []byte{0x01}); err != nil {
		return err
	}
	if err := device.command(0x18, []byte{0x80}); err != nil {
		return err
	}
	if err := device.command(0x1a, []byte{0x5a}); err != nil {
		return err
	}
	return nil
}

func (device *Device) finishGray() error {
	if err := device.command(commandUpdateControl, []byte{0xd7}); err != nil {
		return err
	}
	if err := device.command(commandMasterActivation, nil); err != nil {
		return err
	}
	if err := device.waitReady(); err != nil {
		return err
	}
	return device.deepSleep()
}

func (device *Device) writeGrayPlane(command byte, source GrayPlaneSource, plane int) error {
	if err := device.setRAMWindow(0, 0, Width, Height); err != nil {
		return err
	}
	if err := device.transport.Begin(command); err != nil {
		return err
	}
	var row [BytesPerRow]byte
	for y := 0; y < Height; y++ {
		if err := source.FillGrayRow(plane, y, row[:]); err != nil {
			_ = device.transport.End()
			return err
		}
		if err := device.transport.Data(row[:]); err != nil {
			_ = device.transport.End()
			return err
		}
	}
	return device.transport.End()
}

func (device *Device) resetAndInitializeMonochrome() error {
	if err := device.transport.Reset(); err != nil {
		return err
	}
	if err := device.softwareReset(); err != nil {
		return err
	}
	if err := device.command(0x18, []byte{0x80}); err != nil {
		return err
	}
	if err := device.command(0x0c, []byte{0xae, 0xc7, 0xc3, 0xc0, 0x80}); err != nil {
		return err
	}
	if err := device.command(0x01, []byte{0xdf, 0x01, 0x02}); err != nil {
		return err
	}
	if err := device.command(0x3c, []byte{0x01}); err != nil {
		return err
	}
	if err := device.command(0x21, []byte{0x00}); err != nil {
		return err
	}
	return device.setRAMWindow(0, 0, Width, Height)
}

func (device *Device) softwareReset() error {
	if err := device.waitReady(); err != nil {
		return err
	}
	if err := device.command(commandSoftReset, nil); err != nil {
		return err
	}
	device.transport.DelayMilliseconds(10)
	return device.waitReady()
}

func (device *Device) deepSleep() error {
	if err := device.command(commandDeepSleep, []byte{0x01}); err != nil {
		return err
	}
	device.transport.DelayMilliseconds(100)
	device.fastActive = false
	return nil
}

func (device *Device) writeFastDifferentialLUT() error {
	if err := device.command(commandWriteLUT, fastDifferentialLUT[:105]); err != nil {
		return err
	}
	if err := device.command(commandGateVoltage, fastDifferentialLUT[105:106]); err != nil {
		return err
	}
	if err := device.command(commandSourceVoltage, fastDifferentialLUT[106:109]); err != nil {
		return err
	}
	return device.command(commandWriteVCOM, fastDifferentialLUT[109:])
}

func (device *Device) invalidateAfterFastError(err error) error {
	device.baseline = false
	device.partialRefresh = 0
	device.fastActive = false
	return err
}

func (device *Device) waitReady() error {
	return device.waitReadyWithPoller(nil)
}

func (device *Device) waitReadyWithPoller(poller RefreshPoller) error {
	device.transport.DelayMilliseconds(1)
	started := device.transport.Milliseconds()
	for device.transport.Busy() {
		if device.transport.Milliseconds()-started >= device.busyTimeout {
			return ErrTimeout
		}
		if poller != nil {
			if err := poller.PollDuringRefresh(); err != nil {
				return err
			}
		}
		device.transport.DelayMilliseconds(1)
	}
	return nil
}

func (device *Device) command(command byte, data []byte) error {
	if err := device.transport.Begin(command); err != nil {
		return err
	}
	if len(data) != 0 {
		if err := device.transport.Data(data); err != nil {
			_ = device.transport.End()
			return err
		}
	}
	return device.transport.End()
}

func (device *Device) writeRAM(command byte, frame []byte, invert bool) error {
	if err := device.setRAMWindow(0, 0, Width, Height); err != nil {
		return err
	}
	if err := device.transport.Begin(command); err != nil {
		return err
	}
	if !invert {
		if err := device.transport.Data(frame); err != nil {
			_ = device.transport.End()
			return err
		}
	} else {
		var buffer [64]byte
		for len(frame) != 0 {
			count := len(frame)
			if count > len(buffer) {
				count = len(buffer)
			}
			for index := 0; index < count; index++ {
				buffer[index] = ^frame[index]
			}
			if err := device.transport.Data(buffer[:count]); err != nil {
				_ = device.transport.End()
				return err
			}
			frame = frame[count:]
		}
	}
	return device.transport.End()
}

func (device *Device) setRAMWindow(x, y, width, height int) error {
	xEnd := x + width - 1
	yEnd := y + height - 1
	// The PaperMono panel's gates are wired in reverse order. Keep framebuffer
	// rows in visible top-to-bottom order by incrementing X while decrementing
	// the controller Y counter, matching M5Stack's SSD1677 panel driver.
	if err := device.command(commandDataEntryMode, []byte{0x01}); err != nil {
		return err
	}
	if err := device.command(commandRAMXRange, little16(x, xEnd)); err != nil {
		return err
	}
	if err := device.command(commandRAMYRange, little16(yEnd, y)); err != nil {
		return err
	}
	if err := device.command(commandRAMXCounter, little16(x)); err != nil {
		return err
	}
	return device.command(commandRAMYCounter, little16(yEnd))
}

func little16(values ...int) []byte {
	data := make([]byte, len(values)*2)
	for index, value := range values {
		data[index*2] = byte(value)
		data[index*2+1] = byte(value >> 8)
	}
	return data
}

type protocolError string

func (err protocolError) Error() string { return string(err) }

const (
	ErrFrameSize  protocolError = "SSD1677 frame has the wrong size"
	ErrGrayLevel  protocolError = "SSD1677 gray level is out of range"
	ErrNoBaseline protocolError = "SSD1677 partial refresh requires a full monochrome baseline"
	ErrTimeout    protocolError = "SSD1677 BUSY timeout"
)
