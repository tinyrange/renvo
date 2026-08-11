//go:build renvo && linux

// Command renvoflash is a dependency-free ESP ROM-loader client compiled by
// Renvo itself. Keep this implementation within Renvo's supported language and
// standard-library surface: its only host boundary is the native syscall
// intrinsic used for serial I/O and control-line ioctls.
package main

import "unsafe"

const (
	flashOffsetR   = 0x10000
	flashBlockR    = 0x400
	checksumMagicR = 0xef

	sysReadR      = 0
	sysWriteR     = 1
	sysOpenR      = 2
	sysCloseR     = 3
	sysIoctlR     = 16
	sysNanosleepR = 35

	oReadWriteR = 2
	oNoCTTYR    = 0x100
	tcgetsR     = 0x5401
	tcsetsR     = 0x5402
	tiocmbisR   = 0x5416
	tiocmbicR   = 0x5417
	tiocmDTRR   = 0x002
	tiocmRTSR   = 0x004

	s3Option1R       = 0x6000812c
	s3WDTConfig0R    = 0x60008098
	s3WDTConfig1R    = 0x6000809c
	s3WDTWriteGuardR = 0x600080b0
	s3WDTKeyR        = 0x50d83aa1
)

type flashErrorR struct{ text string }

func (e *flashErrorR) Error() string {
	if e == nil {
		return ""
	}
	return e.text
}

func failR(text string) error { return &flashErrorR{text: text} }

type targetR struct {
	name        string
	machine     uint16
	chipID      uint16
	flashConfig byte
	magic       uint32
}

var targetsR = []targetR{
	{name: "esp32c6/riscv32", machine: 243, chipID: 13, flashConfig: 0x20, magic: 0x2ce0806f},
	{name: "esp32s3/xtensa_lx7", machine: 94, chipID: 9, flashConfig: 0x3f, magic: 0x09},
}

func appMain(args []string, env []string) int {
	if len(args) == 4 && args[1] == "--convert" {
		source, err := readFileR(args[2])
		if err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		image, _, err := prepareImageR(source)
		if err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		if err = writeFileR(args[3], image); err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		return 0
	}
	if len(args) < 2 || len(args) > 3 {
		print("usage: renvoflash ELF-OR-BIN [PORT]\n       renvoflash --convert ELF BIN\n")
		return 2
	}
	port := "/dev/ttyACM0"
	if len(args) == 3 {
		port = args[2]
	}
	if err := flashFileR(args[1], port); err != nil {
		print("renvoflash: " + err.Error() + "\n")
		return 1
	}
	return 0
}

func flashFileR(path string, portName string) error {
	source, err := readFileR(path)
	if err != nil {
		return err
	}
	image, chip, err := prepareImageR(source)
	if err != nil {
		return err
	}
	print("Prepared " + decimalR(len(image)) + " byte ESP app image (" + chip.name + ")\n")
	port, err := openSerialR(portName)
	if err != nil {
		return err
	}
	defer port.close()
	loader := &loaderR{port: port}
	if err := loader.connect(); err != nil {
		return err
	}
	magic, err := loader.command(0x0a, wordsR([]uint32{0x40001000}), 0, 30)
	if err != nil {
		return err
	}
	if magic != chip.magic {
		return failR("connected chip 0x" + hexR(magic) + " does not match " + chip.name)
	}
	print("ROM loader connected (" + chip.name + ")\n")
	if _, err = loader.command(0x0d, wordsR([]uint32{0}), 0, 30); err != nil {
		return err
	}
	blocks := (len(image) + flashBlockR - 1) / flashBlockR
	begin := []uint32{uint32(len(image)), uint32(blocks), flashBlockR, flashOffsetR, 0}
	if _, err = loader.command(0x02, wordsR(begin), 0, 100); err != nil {
		return err
	}
	for sequence := 0; sequence < blocks; sequence++ {
		block := make([]byte, flashBlockR)
		for i := 0; i < len(block); i++ {
			block[i] = 0xff
		}
		start := sequence * flashBlockR
		end := start + flashBlockR
		if end > len(image) {
			end = len(image)
		}
		copy(block, image[start:end])
		checksum := byte(checksumMagicR)
		for i := 0; i < len(block); i++ {
			checksum = checksum ^ block[i]
		}
		payload := wordsR([]uint32{flashBlockR, uint32(sequence), 0, 0})
		payload = append(payload, block...)
		if _, err = loader.command(0x03, payload, uint32(checksum), 50); err != nil {
			return failR("write block " + decimalR(sequence+1) + "/" + decimalR(blocks) + ": " + err.Error())
		}
		if sequence+1 == blocks || (sequence+1)%16 == 0 {
			print("Wrote " + decimalR(sequence+1) + "/" + decimalR(blocks) + " blocks\n")
		}
	}
	if chip.chipID == 9 {
		if err = loader.resetS3Watchdog(); err != nil {
			return err
		}
		print("Application started\n")
		return nil
	}
	if _, err = loader.command(0x04, wordsR([]uint32{1}), 0, 30); err != nil {
		return err
	}
	if err = port.setDTR(false); err != nil {
		return err
	}
	if err = port.setRTS(true); err != nil {
		return err
	}
	sleepR(200)
	// Native USB boards disappear as soon as reset is asserted. Deassertion is
	// best-effort because the old tty may already be gone and the new USB
	// personality may not provide modem-control ioctls.
	_ = port.setRTS(false)
	_ = port.setDTR(false)
	print("Application started\n")
	// Keep the native USB endpoint open for one second while reset settles and
	// relay any boot output, matching the browser workflow.
	buffer := make([]byte, 4096)
	empty := 0
	for empty < 10 {
		n := port.read(buffer)
		if n < 0 {
			return failR("serial monitor read failed")
		}
		if n == 0 {
			empty++
		} else {
			empty = 0
			print(string(buffer[:n]))
		}
	}
	return nil
}

func (loader *loaderR) writeRegister(address uint32, value uint32, mask uint32) error {
	_, err := loader.command(0x09, wordsR([]uint32{address, value, mask, 0}), 0, 30)
	return err
}

// resetS3Watchdog leaves native USB download mode without depending on modem
// control lines that disappear as soon as the shared USB PHY changes owner.
func (loader *loaderR) resetS3Watchdog() error {
	if err := loader.writeRegister(s3Option1R, 0, 1); err != nil {
		return failR("clear S3 forced-download state: " + err.Error())
	}
	if err := loader.writeRegister(s3WDTWriteGuardR, s3WDTKeyR, 0xffffffff); err != nil {
		return failR("unlock S3 watchdog: " + err.Error())
	}
	if err := loader.writeRegister(s3WDTConfig1R, 2000, 0xffffffff); err != nil {
		return failR("set S3 watchdog timeout: " + err.Error())
	}
	if err := loader.writeRegister(s3WDTConfig0R, 0xd0000102, 0xffffffff); err != nil {
		return failR("arm S3 watchdog: " + err.Error())
	}
	if err := loader.writeRegister(s3WDTWriteGuardR, 0, 0xffffffff); err != nil {
		return failR("lock S3 watchdog: " + err.Error())
	}
	sleepR(500)
	return nil
}

type sectionR struct {
	name    string
	address uint32
	data    []byte
}

func prepareImageR(source []byte) ([]byte, targetR, error) {
	if len(source) >= 4 && source[0] == 0x7f && source[1] == 'E' && source[2] == 'L' && source[3] == 'F' {
		return elfToImageR(source)
	}
	if len(source) < 24 || source[0] != 0xe9 {
		return nil, targetR{}, failR("input is neither an ELF32 file nor an ESP application image")
	}
	chipID := u16R(source, 12)
	for i := 0; i < len(targetsR); i++ {
		if targetsR[i].chipID == chipID {
			return source, targetsR[i], nil
		}
	}
	return nil, targetR{}, failR("unsupported ESP image chip ID " + decimalR(int(chipID)))
}

func elfToImageR(source []byte) ([]byte, targetR, error) {
	if len(source) < 52 || source[4] != 1 || source[5] != 1 {
		return nil, targetR{}, failR("ESP flashing requires a little-endian ELF32 file")
	}
	machine := u16R(source, 18)
	chip := targetR{}
	for i := 0; i < len(targetsR); i++ {
		if targetsR[i].machine == machine {
			chip = targetsR[i]
		}
	}
	if chip.name == "" {
		return nil, targetR{}, failR("unsupported ELF machine " + decimalR(int(machine)))
	}
	entry := u32R(source, 24)
	sectionAt := int(u32R(source, 32))
	sectionSize := int(u16R(source, 46))
	sectionCount := int(u16R(source, 48))
	namesIndex := int(u16R(source, 50))
	if sectionSize < 40 || namesIndex >= sectionCount || sectionAt < 0 || sectionAt+sectionSize*sectionCount > len(source) {
		return nil, targetR{}, failR("ELF section table is invalid")
	}
	namesHeader := sectionAt + namesIndex*sectionSize
	namesAt := int(u32R(source, namesHeader+16))
	namesSize := int(u32R(source, namesHeader+20))
	if namesAt < 0 || namesAt+namesSize > len(source) {
		return nil, targetR{}, failR("ELF section names are invalid")
	}
	var appdesc []sectionR
	var ram []sectionR
	var flash []sectionR
	for index := 0; index < sectionCount; index++ {
		at := sectionAt + index*sectionSize
		typ := u32R(source, at+4)
		flags := u32R(source, at+8)
		offset := int(u32R(source, at+16))
		size := int(u32R(source, at+20))
		if typ != 1 || flags&2 == 0 || size == 0 {
			continue
		}
		if offset < 0 || offset+size > len(source) {
			return nil, targetR{}, failR("ELF section data is invalid")
		}
		nameOffset := int(u32R(source, at))
		if nameOffset < 0 || nameOffset >= namesSize {
			return nil, targetR{}, failR("ELF section name is invalid")
		}
		nameEnd := namesAt + nameOffset
		for nameEnd < namesAt+namesSize && source[nameEnd] != 0 {
			nameEnd++
		}
		var item sectionR
		item.name = string(source[namesAt+nameOffset : nameEnd])
		item.address = u32R(source, at+12)
		item.data = append([]byte{}, source[offset:offset+size]...)
		if item.name == ".flash.appdesc" {
			appdesc = append(appdesc, item)
		} else if flashAddressR(chip, item.address) {
			flash = append(flash, item)
		} else {
			ram = append(ram, item)
		}
	}
	sortSectionsR(ram)
	sortSectionsR(flash)
	segments := make([]sectionR, 0, len(appdesc)+len(ram)+len(flash)*2)
	imageLength := 24
	for i := 0; i < len(appdesc); i++ {
		segments, imageLength = appendSegmentR(segments, imageLength, appdesc[i])
	}
	for i := 0; i < len(ram); i++ {
		segments, imageLength = appendSegmentR(segments, imageLength, ram[i])
	}
	for i := 0; i < len(flash); i++ {
		desired := int((flash[i].address - flashOffsetR) & 0xffff)
		if (imageLength+8)&0xffff != desired {
			padding := (desired - imageLength - 16) & 0xffff
			segments, imageLength = appendSegmentR(segments, imageLength, sectionR{data: make([]byte, padding)})
		}
		segments, imageLength = appendSegmentR(segments, imageLength, flash[i])
	}
	if len(segments) == 0 || len(segments) > 255 {
		return nil, targetR{}, failR("ELF has an invalid number of loadable sections")
	}
	image := []byte{0xe9, byte(len(segments)), 0x02, chip.flashConfig}
	image = append(image, wordR(entry)...)
	// hash_appended is zero. The ESP image checksum below remains mandatory;
	// omitting the optional SHA-256 trailer keeps this Renvo-only utility small
	// and avoids depending on a crypto standard-library package.
	image = append(image, 0xee, 0, 0, 0, byte(chip.chipID), byte(chip.chipID>>8), 0, 0, 0, 0xff, 0xff, 0, 0, 0, 0, 0)
	checksum := byte(checksumMagicR)
	for i := 0; i < len(segments); i++ {
		image = append(image, wordR(segments[i].address)...)
		image = append(image, wordR(uint32(len(segments[i].data)))...)
		image = append(image, segments[i].data...)
		for j := 0; j < len(segments[i].data); j++ {
			checksum = checksum ^ segments[i].data[j]
		}
	}
	for len(image)&15 != 15 {
		image = append(image, 0)
	}
	image = append(image, checksum)
	return image, chip, nil
}

func flashAddressR(chip targetR, address uint32) bool {
	if chip.chipID == 13 {
		return address >= 0x42000000 && address < 0x43000000
	}
	return address >= 0x3c000000 && address < 0x3d000000 || address >= 0x42000000 && address < 0x44000000
}

func appendSegmentR(segments []sectionR, imageLength int, item sectionR) ([]sectionR, int) {
	for len(item.data)&3 != 0 {
		item.data = append(item.data, 0)
	}
	segments = append(segments, item)
	return segments, imageLength + 8 + len(item.data)
}

func sortSectionsR(items []sectionR) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].address < items[j-1].address; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

type serialR struct{ fd int }

func openSerialR(path string) (*serialR, error) {
	name := append([]byte(path), 0)
	fd := syscall(sysOpenR, int(unsafe.Pointer(&name[0])), oReadWriteR|oNoCTTYR, 0, 0, 0, 0)
	if fd < 0 {
		return nil, failR("open serial port " + path + " failed")
	}
	port := &serialR{fd: fd}
	termios := make([]byte, 64)
	result := port.ioctl(tcgetsR, termios)
	if result < 0 {
		port.close()
		return nil, failR("read serial settings failed (errno " + decimalR(-result) + ")")
	}
	put32R(termios, 0, 4)                      // IGNPAR
	put32R(termios, 4, 0)                      // no output processing
	put32R(termios, 8, 0x1002|0x30|0x80|0x800) // B115200, CS8, CREAD, CLOCAL
	put32R(termios, 12, 0)                     // raw local mode
	termios[17+5] = 1                          // VTIME: 100 ms
	termios[17+6] = 0                          // VMIN
	result = port.ioctl(tcsetsR, termios)
	if result < 0 {
		port.close()
		return nil, failR("configure serial port failed (errno " + decimalR(-result) + ")")
	}
	return port, nil
}

func (port *serialR) ioctl(request int, data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return syscall(sysIoctlR, port.fd, request, int(unsafe.Pointer(&data[0])), 0, 0, 0)
}

func (port *serialR) close() { syscall(sysCloseR, port.fd, 0, 0, 0, 0, 0) }

func (port *serialR) setDTR(state bool) error { return port.modem(tiocmDTRR, state) }
func (port *serialR) setRTS(state bool) error { return port.modem(tiocmRTSR, state) }

func (port *serialR) modem(bit int, state bool) error {
	data := make([]byte, 4)
	put32R(data, 0, bit)
	request := tiocmbicR
	if state {
		request = tiocmbisR
	}
	if port.ioctl(request, data) < 0 {
		return failR("set serial control lines failed")
	}
	return nil
}

func (port *serialR) write(data []byte) error {
	offset := 0
	for offset < len(data) {
		n := syscall(sysWriteR, port.fd, int(unsafe.Pointer(&data[offset])), len(data)-offset, 0, 0, 0)
		if n <= 0 {
			return failR("serial write failed")
		}
		offset += n
	}
	return nil
}

func (port *serialR) read(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	return syscall(sysReadR, port.fd, int(unsafe.Pointer(&data[0])), len(data), 0, 0, 0)
}

type loaderR struct {
	port    *serialR
	frame   []byte
	escaped bool
	inside  bool
}

func (loader *loaderR) connect() error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		if attempt == 0 {
			err = loader.usbReset()
		} else if attempt == 1 {
			err = loader.classicReset(50)
		} else {
			err = loader.classicReset(550)
		}
		if err != nil {
			last = err
			continue
		}
		loader.frame = nil
		loader.inside = false
		loader.escaped = false
		if err = loader.sync(); err == nil {
			return nil
		}
		last = err
	}
	return failR("could not synchronize with ESP ROM loader: " + last.Error())
}

func (loader *loaderR) usbReset() error {
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	sleepR(100)
	if err := loader.port.setDTR(true); err != nil {
		return err
	}
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	sleepR(100)
	if err := loader.port.setRTS(true); err != nil {
		return err
	}
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	sleepR(100)
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	return loader.port.setDTR(false)
}

func (loader *loaderR) classicReset(delay int) error {
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	if err := loader.port.setRTS(true); err != nil {
		return err
	}
	sleepR(100)
	if err := loader.port.setDTR(true); err != nil {
		return err
	}
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	sleepR(delay)
	return loader.port.setDTR(false)
}

func (loader *loaderR) sync() error {
	payload := make([]byte, 36)
	payload[0], payload[1], payload[2], payload[3] = 7, 7, 0x12, 0x20
	for i := 4; i < len(payload); i++ {
		payload[i] = 0x55
	}
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := loader.command(0x08, payload, 0, 3); err == nil {
			return nil
		} else {
			last = err
		}
		sleepR(80)
	}
	return last
}

func (loader *loaderR) command(opcode byte, payload []byte, checksum uint32, emptyLimit int) (uint32, error) {
	request := []byte{0, opcode, byte(len(payload)), byte(len(payload) >> 8)}
	request = append(request, wordR(checksum)...)
	request = append(request, payload...)
	if err := loader.port.write(slipR(request)); err != nil {
		return 0, err
	}
	for empty := 0; empty < emptyLimit; {
		frame, hadData, err := loader.nextFrame()
		if err != nil {
			return 0, err
		}
		if !hadData {
			empty++
			continue
		}
		if len(frame) == 0 {
			continue
		}
		if len(frame) < 8 || frame[0] != 1 || frame[1] != opcode {
			continue
		}
		body := frame[8:]
		if len(body) >= 2 && body[len(body)-2] != 0 {
			return 0, failR("ESP command 0x" + hexR(uint32(opcode)) + " failed")
		}
		return u32R(frame, 4), nil
	}
	return 0, failR("ESP command 0x" + hexR(uint32(opcode)) + " timed out")
}

func (loader *loaderR) nextFrame() ([]byte, bool, error) {
	buffer := make([]byte, 256)
	n := loader.port.read(buffer)
	if n < 0 {
		return nil, false, failR("serial read failed")
	}
	if n == 0 {
		return nil, false, nil
	}
	for i := 0; i < n; i++ {
		value := buffer[i]
		if value == 0xc0 {
			if loader.inside && len(loader.frame) != 0 {
				result := append([]byte{}, loader.frame...)
				loader.frame = loader.frame[:0]
				loader.escaped = false
				return result, true, nil
			}
			loader.inside = true
			loader.frame = loader.frame[:0]
			loader.escaped = false
		} else if !loader.inside {
			continue
		} else if loader.escaped {
			if value == 0xdc {
				loader.frame = append(loader.frame, 0xc0)
			} else if value == 0xdd {
				loader.frame = append(loader.frame, 0xdb)
			} else {
				loader.frame = append(loader.frame, 0xdb, value)
			}
			loader.escaped = false
		} else if value == 0xdb {
			loader.escaped = true
		} else {
			loader.frame = append(loader.frame, value)
		}
	}
	return nil, true, nil
}

func slipR(data []byte) []byte {
	result := []byte{0xc0}
	for i := 0; i < len(data); i++ {
		if data[i] == 0xc0 {
			result = append(result, 0xdb, 0xdc)
		} else if data[i] == 0xdb {
			result = append(result, 0xdb, 0xdd)
		} else {
			result = append(result, data[i])
		}
	}
	return append(result, 0xc0)
}

func readFileR(path string) ([]byte, error) {
	name := append([]byte(path), 0)
	fd := syscall(sysOpenR, int(unsafe.Pointer(&name[0])), 0, 0, 0, 0, 0)
	if fd < 0 {
		return nil, failR("read " + path + " failed")
	}
	defer syscall(sysCloseR, fd, 0, 0, 0, 0, 0)
	var result []byte
	buffer := make([]byte, 4096)
	for {
		n := syscall(sysReadR, fd, int(unsafe.Pointer(&buffer[0])), len(buffer), 0, 0, 0)
		if n < 0 {
			return nil, failR("read " + path + " failed")
		}
		if n == 0 {
			return result, nil
		}
		result = append(result, buffer[:n]...)
	}
}

func writeFileR(path string, data []byte) error {
	name := append([]byte(path), 0)
	fd := syscall(sysOpenR, int(unsafe.Pointer(&name[0])), 2|64|512, 0x1b6, 0, 0, 0)
	if fd < 0 {
		return failR("write " + path + " failed")
	}
	offset := 0
	for offset < len(data) {
		n := syscall(sysWriteR, fd, int(unsafe.Pointer(&data[offset])), len(data)-offset, 0, 0, 0)
		if n <= 0 {
			syscall(sysCloseR, fd, 0, 0, 0, 0, 0)
			return failR("write " + path + " failed")
		}
		offset += n
	}
	syscall(sysCloseR, fd, 0, 0, 0, 0, 0)
	return nil
}

func sleepR(milliseconds int) {
	timespec := make([]byte, 16)
	seconds := milliseconds / 1000
	nanoseconds := (milliseconds % 1000) * 1000000
	put64R(timespec, 0, int64(seconds))
	put64R(timespec, 8, int64(nanoseconds))
	syscall(sysNanosleepR, int(unsafe.Pointer(&timespec[0])), 0, 0, 0, 0, 0)
}

func u16R(data []byte, at int) uint16 { return uint16(data[at]) | uint16(data[at+1])<<8 }
func u32R(data []byte, at int) uint32 {
	return uint32(data[at]) | uint32(data[at+1])<<8 | uint32(data[at+2])<<16 | uint32(data[at+3])<<24
}
func wordR(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}
func wordsR(values []uint32) []byte {
	var out []byte
	for i := 0; i < len(values); i++ {
		out = append(out, wordR(values[i])...)
	}
	return out
}
func put32R(data []byte, at int, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
	data[at+2] = byte(value >> 16)
	data[at+3] = byte(value >> 24)
}
func put64R(data []byte, at int, value int64) {
	for i := 0; i < 8; i++ {
		data[at+i] = byte(value >> uint(i*8))
	}
}
func decimalR(value int) string {
	if value == 0 {
		return "0"
	}
	var reverse []byte
	for value > 0 {
		reverse = append(reverse, byte('0'+value%10))
		value /= 10
	}
	var result []byte
	for i := len(reverse) - 1; i >= 0; i-- {
		result = append(result, reverse[i])
	}
	return string(result)
}
func hexR(value uint32) string {
	if value == 0 {
		return "0"
	}
	var reverse []byte
	for value != 0 {
		reverse = append(reverse, "0123456789abcdef"[value&15])
		value >>= 4
	}
	var result []byte
	for i := len(reverse) - 1; i >= 0; i-- {
		result = append(result, reverse[i])
	}
	return string(result)
}
