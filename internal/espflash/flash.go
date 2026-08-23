//go:build linux || darwin

// Command renvoflash is a dependency-free ESP ROM-loader client compiled by
// Renvo itself. Keep this implementation within Renvo's supported language and
// standard-library surface: its only host boundary is the native syscall
// intrinsic used for serial I/O and control-line ioctls.
package espflash

import (
	"os"
)

const (
	flashOffset   = 0x10000
	flashBlock    = 0x400
	flashBatch    = 128
	checksumMagic = 0xef
	backupChunk   = 64
	backupBuffer  = 64 * 1024
	backupBatch   = 32
)

type flashError struct{ text string }

func (e *flashError) Error() string {
	if e == nil {
		return ""
	}
	return e.text
}

func fail(text string) error { return &flashError{text: text} }

type target struct {
	name        string
	machine     uint16
	chipID      uint16
	flashConfig byte
	magic       uint32
	flashSize   int
}

// Image identifies a converted ESP application image.
type Image struct {
	Data   []byte
	Target string
}

// Prepare converts ELF32 input when necessary and validates the ESP target.
func Prepare(source []byte) (Image, error) {
	data, chip, err := prepareImage(source)
	if err != nil {
		return Image{}, err
	}
	return Image{Data: data, Target: chip.name}, nil
}

// Flash writes an ELF or application image through the ESP ROM loader.
func Flash(path string, port string) error { return flashFile(path, port) }

// Backup creates a read-only whole-flash backup through the ESP ROM loader.
func Backup(path string, port string) error { return backupFlash(path, port) }

// DetectSerialPort returns the only supported USB serial endpoint.
func DetectSerialPort() (string, error) { return detectPort() }

var targets = []target{
	{name: "esp32c6/riscv32", machine: 243, chipID: 13, flashConfig: 0x20, magic: 0x2ce0806f, flashSize: 4 * 1024 * 1024},
	{name: "esp32s3/xtensa_lx7", machine: 94, chipID: 9, flashConfig: 0x3f, magic: 0x09, flashSize: 8 * 1024 * 1024},
	// ESP32-P4 shares ELF machine 243 with ESP32-C6. P4 application binaries
	// are accepted by chip ID, but ELF conversion remains C6 until a P4 board
	// target can supply an unambiguous image target.
	{name: "esp32p4/riscv32", machine: 243, chipID: 18, flashConfig: 0x4f, flashSize: 16 * 1024 * 1024},
}

// Run executes the renvoflash command. It is exported so the Renvo frontend
// can embed the flasher without spawning another process.
func Run(args []string, env []string) int {
	if handled, status := runJTAGCommand(args); handled {
		return status
	}
	if len(args) >= 3 && len(args) <= 4 && args[1] == "--backup" {
		port := ""
		if len(args) == 4 {
			port = args[3]
		} else {
			var detectErr error
			port, detectErr = detectPort()
			if detectErr != nil {
				print("renvoflash: " + detectErr.Error() + "\n")
				return 1
			}
		}
		if err := backupFlash(args[2], port); err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		return 0
	}
	if len(args) == 4 && args[1] == "--convert" {
		source, err := readFile(args[2])
		if err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		image, _, err := prepareImage(source)
		if err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		if err = writeFile(args[3], image); err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return 1
		}
		return 0
	}
	if len(args) < 2 || len(args) > 3 {
		print("usage: renvoflash ELF-OR-BIN [PORT]\n       renvoflash --backup BIN [PORT]\n       renvoflash --convert ELF BIN\n       renvoflash --probe-jtag [USB-DEVICE]\n       renvoflash --jtag ELF [USB-DEVICE]\n       renvoflash --watch ELF [USB-DEVICE]\n       renvoflash --jtag-status|--jtag-halt|--jtag-resume|--jtag-regs|--jtag-step [USB-DEVICE]\n       renvoflash --jtag-read ADDRESS SIZE [USB-DEVICE]\n       renvoflash --jtag-set-pc ADDRESS [USB-DEVICE]\n       renvoflash --jtag-set-reg REGISTER VALUE [USB-DEVICE]\n       renvoflash --jtag-blink CYCLES HALF-PERIOD-MS [USB-DEVICE]\n")
		return 2
	}
	port := ""
	if len(args) == 3 {
		port = args[2]
	} else {
		var detectErr error
		port, detectErr = detectPort()
		if detectErr != nil {
			print("renvoflash: " + detectErr.Error() + "\n")
			return 1
		}
	}
	if err := flashFile(args[1], port); err != nil {
		print("renvoflash: " + err.Error() + "\n")
		return 1
	}
	return 0
}

func detectPort() (string, error) {
	entries, err := os.ReadDir("/dev")
	if err != nil {
		return "", fail("could not scan /dev for a serial port")
	}
	prefixes := serialPortPrefixes()
	var matches []string
	for i := 0; i < len(entries); i++ {
		name := entries[i].Name()
		for j := 0; j < len(prefixes); j++ {
			if hasPrefix(name, prefixes[j]) {
				matches = append(matches, "/dev/"+name)
				break
			}
		}
	}
	if len(matches) == 0 {
		return "", fail("no USB serial port found; connect the board or pass PORT explicitly")
	}
	if len(matches) != 1 {
		message := "multiple USB serial ports found"
		for i := 0; i < len(matches); i++ {
			message += " " + matches[i]
		}
		return "", fail(message + "; pass PORT explicitly")
	}
	print("Detected serial port " + matches[0] + "\n")
	return matches[0], nil
}

func hasPrefix(text string, prefix string) bool {
	if len(text) < len(prefix) {
		return false
	}
	return text[:len(prefix)] == prefix
}

func backupFlash(path string, portName string) error {
	port, err := openSerial(portName)
	if err != nil {
		return err
	}
	defer port.close()
	loader := &loader{port: port}
	print("Serial port configured\n")
	if err = loader.connect(); err != nil {
		return err
	}
	chip, security, err := loader.identify()
	if err != nil {
		return err
	}
	print("ROM loader connected (" + chip.name + ")\n")
	if len(security) >= 20 {
		flags := u32(security, 0)
		crypt := security[4]
		print("Security flags 0x" + hex(flags) + ", flash crypt count " + decimal(int(crypt)) + "\n")
		if flags&4 != 0 {
			return fail("secure download mode does not permit a restorable firmware backup")
		}
		if bitCount(crypt)&1 != 0 {
			return fail("flash encryption is enabled; refusing to create a plaintext backup")
		}
	}
	if _, err = loader.command(0x0d, words([]uint32{0}), 0, 30); err != nil {
		return err
	}
	flashParameters := words([]uint32{0, uint32(chip.flashSize), 64 * 1024, 4 * 1024, 256, 0xffff})
	if _, err = loader.command(0x0b, flashParameters, 0, 30); err != nil {
		return fail("configure flash geometry: " + err.Error())
	}
	digestPayload := words([]uint32{0, uint32(chip.flashSize), 0, 0})
	digest, digestErr := loader.commandData(0x13, digestPayload, 0, 30000)
	if digestErr != nil {
		return fail("device digest failed: " + digestErr.Error())
	}
	if len(digest) < 32 {
		return fail("device returned an invalid flash digest")
	}
	fd := hostOpen(path, hostCreateFlags(), 0x1b6)
	if fd < 0 {
		return fail("write " + path + " failed")
	}
	completed := false
	defer func() {
		hostClose(fd)
		if !completed {
			print("Backup is incomplete; do not use it for restore\n")
		}
	}()
	print("Reading " + decimal(chip.flashSize) + " bytes to " + path + "\n")
	payload := make([]byte, 8)
	block := make([]byte, backupChunk)
	buffer := make([]byte, 0, backupBuffer)
	batchData := make([]byte, backupBatch*backupChunk)
	for batchOffset := 0; batchOffset < chip.flashSize; batchOffset += backupChunk * backupBatch {
		batchCount := backupBatch
		if batchOffset+batchCount*backupChunk > chip.flashSize {
			batchCount = (chip.flashSize - batchOffset) / backupChunk
		}
		for i := 0; i < batchCount; i++ {
			put32(payload, 0, batchOffset+i*backupChunk)
			put32(payload, 4, backupChunk)
			if commandErr := loader.sendCommand(0x0e, payload, 0); commandErr != nil {
				return fail("request flash at 0x" + hex(uint32(batchOffset+i*backupChunk)) + ": " + commandErr.Error())
			}
		}
		if commandErr := loader.receiveFlashBatch(batchCount, batchData[:batchCount*backupChunk]); commandErr != nil {
			return fail("read flash batch at 0x" + hex(uint32(batchOffset)) + ": " + commandErr.Error())
		}
		for i := 0; i < batchCount; i++ {
			offset := batchOffset + i*backupChunk
			copy(block, batchData[i*backupChunk:(i+1)*backupChunk])
			buffer = append(buffer, block...)
			if len(buffer) == backupBuffer {
				if hostWriteAll(fd, buffer) < 0 {
					return fail("write " + path + " failed")
				}
				buffer = buffer[:0]
			}
			read := offset + backupChunk
			if read%(1024*1024) == 0 || read == chip.flashSize {
				print("Read " + decimal(read/(1024*1024)) + "/" + decimal(chip.flashSize/(1024*1024)) + " MiB\n")
			}
		}
	}
	if len(buffer) != 0 && hostWriteAll(fd, buffer) < 0 {
		return fail("write " + path + " failed")
	}
	completed = true
	print("Backup complete\nDevice MD5: " + string(digest[:32]) + "\n")
	return nil
}

func flashFile(path string, portName string) error {
	source, err := readFile(path)
	if err != nil {
		return err
	}
	image, chip, err := prepareImage(source)
	if err != nil {
		return err
	}
	print("Prepared " + decimal(len(image)) + " byte ESP app image (" + chip.name + ")\n")
	port, err := openSerial(portName)
	if err != nil {
		return err
	}
	defer port.close()
	loader := &loader{port: port}
	if err := loader.connect(); err != nil {
		return err
	}
	connected, _, err := loader.identify()
	if err != nil {
		return err
	}
	if connected.chipID != chip.chipID {
		return fail("connected " + connected.name + " does not match " + chip.name)
	}
	print("ROM loader connected (" + chip.name + ")\n")
	if _, err = loader.command(0x0d, words([]uint32{0}), 0, 30); err != nil {
		return err
	}
	blocks := (len(image) + flashBlock - 1) / flashBlock
	written := 0
	// Bound each ROM erase so native USB never disappears behind a long
	// FLASH_BEGIN. Write high addresses first and the image-header batch last;
	// an interrupted update therefore remains explicitly non-bootable.
	for batchEnd := blocks; batchEnd > 0; {
		batchStart := (batchEnd - 1) / flashBatch * flashBatch
		batchBlocks := batchEnd - batchStart
		begin := []uint32{
			uint32(batchBlocks * flashBlock), uint32(batchBlocks), flashBlock,
			uint32(flashOffset + batchStart*flashBlock), 0,
		}
		if _, err = loader.command(0x02, words(begin), 0, 10000); err != nil {
			return fail("erase batch at 0x" + hex(uint32(flashOffset+batchStart*flashBlock)) + ": " + err.Error())
		}
		for sequence := 0; sequence < batchBlocks; sequence++ {
			blockIndex := batchStart + sequence
			block := make([]byte, flashBlock)
			for i := 0; i < len(block); i++ {
				block[i] = 0xff
			}
			start := blockIndex * flashBlock
			end := start + flashBlock
			if end > len(image) {
				end = len(image)
			}
			copy(block, image[start:end])
			checksum := byte(checksumMagic)
			for i := 0; i < len(block); i++ {
				checksum = checksum ^ block[i]
			}
			payload := words([]uint32{flashBlock, uint32(sequence), 0, 0})
			payload = append(payload, block...)
			if _, err = loader.command(0x03, payload, uint32(checksum), 50); err != nil {
				return fail("write block " + decimal(blockIndex+1) + "/" + decimal(blocks) + ": " + err.Error())
			}
			written++
			if written == blocks || written%16 == 0 {
				print("Wrote " + decimal(written) + "/" + decimal(blocks) + " blocks\n")
			}
		}
		batchEnd = batchStart
	}
	// Finalize the flash while remaining in the loader so the response proves
	// that the last write has completed. FLASH_END terminates the current flash
	// operation even when its stay-in-loader flag is set, so a second FLASH_END
	// is not a reliable way to request the reboot. Use the USB Serial/JTAG hard
	// reset sequence from Espressif's host tools instead.
	if _, err = loader.command(0x04, words([]uint32{1}), 0, 30); err != nil {
		return err
	}
	if err = port.setDTR(false); err != nil {
		return err
	}
	if err = port.setRTS(true); err != nil {
		return err
	}
	// Native USB needs time to process the control-line change on each side of
	// reset before accepting the next one.
	sleep(200)
	if err = port.setRTS(false); err != nil {
		return err
	}
	sleep(200)
	if err = port.setDTR(false); err != nil {
		return err
	}
	print("Application started\n")
	// Native USB disappears during the reboot. Reopen it for the short serial
	// monitor when the application exposes a serial endpoint, but flashing is
	// already complete if an application deliberately does not do so.
	if err = port.reopen(10); err != nil {
		return nil
	}
	buffer := make([]byte, 4096)
	empty := 0
	for empty < 10 {
		n := port.read(buffer)
		if n < 0 {
			return fail("serial monitor read failed")
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

type section struct {
	name    string
	address uint32
	data    []byte
}

func prepareImage(source []byte) ([]byte, target, error) {
	if len(source) >= 4 && source[0] == 0x7f && source[1] == 'E' && source[2] == 'L' && source[3] == 'F' {
		return elfToImage(source)
	}
	if len(source) < 24 || source[0] != 0xe9 {
		return nil, target{}, fail("input is neither an ELF32 file nor an ESP application image")
	}
	chipID := u16(source, 12)
	for i := 0; i < len(targets); i++ {
		if targets[i].chipID == chipID {
			return source, targets[i], nil
		}
	}
	return nil, target{}, fail("unsupported ESP image chip ID " + decimal(int(chipID)))
}

func elfToImage(source []byte) ([]byte, target, error) {
	if len(source) < 52 || source[4] != 1 || source[5] != 1 {
		return nil, target{}, fail("ESP flashing requires a little-endian ELF32 file")
	}
	machine := u16(source, 18)
	entry := u32(source, 24)
	if machine == 243 && c6RAMAddress(entry, 1) {
		return nil, target{}, fail("SRAM-linked ESP32-C6 ELF cannot be converted to a flash image; use --jtag")
	}
	chip := targetForELF(machine, entry)
	if chip.name == "" {
		return nil, target{}, fail("unsupported ELF machine " + decimal(int(machine)))
	}
	sectionAt := int(u32(source, 32))
	sectionSize := int(u16(source, 46))
	sectionCount := int(u16(source, 48))
	namesIndex := int(u16(source, 50))
	if sectionSize < 40 || namesIndex >= sectionCount || sectionAt < 0 || sectionAt+sectionSize*sectionCount > len(source) {
		return nil, target{}, fail("ELF section table is invalid")
	}
	namesHeader := sectionAt + namesIndex*sectionSize
	namesAt := int(u32(source, namesHeader+16))
	namesSize := int(u32(source, namesHeader+20))
	if namesAt < 0 || namesAt+namesSize > len(source) {
		return nil, target{}, fail("ELF section names are invalid")
	}
	var appdesc []section
	var ram []section
	var flash []section
	for index := 0; index < sectionCount; index++ {
		at := sectionAt + index*sectionSize
		typ := u32(source, at+4)
		flags := u32(source, at+8)
		offset := int(u32(source, at+16))
		size := int(u32(source, at+20))
		if typ != 1 || flags&2 == 0 || size == 0 {
			continue
		}
		if offset < 0 || offset+size > len(source) {
			return nil, target{}, fail("ELF section data is invalid")
		}
		nameOffset := int(u32(source, at))
		if nameOffset < 0 || nameOffset >= namesSize {
			return nil, target{}, fail("ELF section name is invalid")
		}
		nameEnd := namesAt + nameOffset
		for nameEnd < namesAt+namesSize && source[nameEnd] != 0 {
			nameEnd++
		}
		var item section
		item.name = string(source[namesAt+nameOffset : nameEnd])
		item.address = u32(source, at+12)
		item.data = append([]byte{}, source[offset:offset+size]...)
		if item.name == ".flash.appdesc" {
			appdesc = append(appdesc, item)
		} else if flashAddress(chip, item.address) {
			flash = append(flash, item)
		} else {
			ram = append(ram, item)
		}
	}
	sortSections(ram)
	sortSections(flash)
	segments := make([]section, 0, len(appdesc)+len(ram)+len(flash)*2)
	imageLength := 24
	for i := 0; i < len(appdesc); i++ {
		segments, imageLength = appendSegment(segments, imageLength, appdesc[i])
	}
	for i := 0; i < len(ram); i++ {
		segments, imageLength = appendSegment(segments, imageLength, ram[i])
	}
	for i := 0; i < len(flash); i++ {
		desired := int((flash[i].address - flashOffset) & 0xffff)
		if (imageLength+8)&0xffff != desired {
			padding := (desired - imageLength - 16) & 0xffff
			segments, imageLength = appendSegment(segments, imageLength, section{data: make([]byte, padding)})
		}
		segments, imageLength = appendSegment(segments, imageLength, flash[i])
	}
	if len(segments) == 0 || len(segments) > 255 {
		return nil, target{}, fail("ELF has an invalid number of loadable sections")
	}
	image := []byte{0xe9, byte(len(segments)), 0x02, chip.flashConfig}
	image = append(image, word(entry)...)
	// hash_appended is zero. The ESP image checksum below remains mandatory;
	// omitting the optional SHA-256 trailer keeps this Renvo-only utility small
	// and avoids depending on a crypto standard-library package.
	image = append(image, 0xee, 0, 0, 0, byte(chip.chipID), byte(chip.chipID>>8), 0, 0, 0, 0xff, 0xff, 0, 0, 0, 0, 0)
	checksum := byte(checksumMagic)
	for i := 0; i < len(segments); i++ {
		image = append(image, word(segments[i].address)...)
		image = append(image, word(uint32(len(segments[i].data)))...)
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

func targetForELF(machine uint16, entry uint32) target {
	// ESP32-C6 and ESP32-P4 both use the standard RISC-V ELF machine ID.
	// Their flash execution windows are distinct in Renvo-produced images.
	if machine == 243 {
		if entry >= 0x42000000 && entry < 0x43000000 {
			return targets[0]
		}
		if entry >= 0x40000000 && entry < 0x44000000 || entry >= 0x4ff00000 && entry < 0x4ffc0000 {
			return targets[2]
		}
	}
	for i := 0; i < len(targets); i++ {
		if targets[i].machine == machine && targets[i].chipID != 13 && targets[i].chipID != 18 {
			return targets[i]
		}
	}
	return target{}
}

func flashAddress(chip target, address uint32) bool {
	if chip.chipID == 13 {
		return address >= 0x42000000 && address < 0x43000000
	}
	if chip.chipID == 18 {
		return address >= 0x40000000 && address < 0x44000000 || address >= 0x48000000 && address < 0x4c000000
	}
	return address >= 0x3c000000 && address < 0x3d000000 || address >= 0x42000000 && address < 0x44000000
}

func appendSegment(segments []section, imageLength int, item section) ([]section, int) {
	for len(item.data)&3 != 0 {
		item.data = append(item.data, 0)
	}
	segments = append(segments, item)
	return segments, imageLength + 8 + len(item.data)
}

func sortSections(items []section) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].address < items[j-1].address; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

type serial struct {
	fd   int
	path string
	dtr  bool
}

func openSerial(path string) (*serial, error) {
	fd := hostOpen(path, hostSerialFlags(), 0)
	if fd < 0 {
		return nil, fail("open serial port " + path + " failed")
	}
	port := &serial{fd: fd, path: path}
	if configureSerial(port) < 0 {
		port.close()
		return nil, fail("configure serial port failed")
	}
	return port, nil
}

func (port *serial) ioctl(request int, data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return hostIoctl(port.fd, request, hostPointer(&data[0]))
}

func (port *serial) close() {
	if port.fd >= 0 {
		hostClose(port.fd)
		port.fd = -1
	}
}

func (port *serial) reopen(attempts int) error {
	port.close()
	for attempt := 0; attempt < attempts; attempt++ {
		fd := hostOpen(port.path, hostSerialFlags(), 0)
		if fd >= 0 {
			port.fd = fd
			if configureSerial(port) >= 0 {
				return nil
			}
			port.close()
		}
		sleep(100)
	}
	return fail("reopen serial port " + port.path + " failed")
}

func (port *serial) setDTR(state bool) error {
	if err := port.modem(hostDTR(), state); err != nil {
		return err
	}
	port.dtr = state
	return nil
}

func (port *serial) setRTS(state bool) error {
	if err := port.modem(hostRTS(), state); err != nil {
		return err
	}
	// Some native USB drivers coalesce SET_CONTROL_LINE_STATE requests. A DTR
	// rewrite after every RTS transition makes each phase observable and is the
	// sequence used by Espressif's host tools on every platform.
	return port.modem(hostDTR(), port.dtr)
}

func (port *serial) setLines(dtr bool, rts bool) error {
	data := make([]byte, 4)
	result := port.ioctl(hostModemGet(), data)
	if result < 0 {
		return fail("read serial control lines failed (host error " + decimal(hostError(result)) + ")")
	}
	bits := int(u32(data, 0))
	if dtr {
		bits = bits | hostDTR()
	} else {
		bits = bits &^ hostDTR()
	}
	if rts {
		bits = bits | hostRTS()
	} else {
		bits = bits &^ hostRTS()
	}
	put32(data, 0, bits)
	result = port.ioctl(hostModemWrite(), data)
	if result < 0 {
		return fail("set serial control lines failed (host error " + decimal(hostError(result)) + ")")
	}
	port.dtr = dtr
	return nil
}

func (port *serial) modem(bit int, state bool) error {
	data := make([]byte, 4)
	put32(data, 0, bit)
	request := hostModemClear()
	if state {
		request = hostModemSet()
	}
	result := port.ioctl(request, data)
	if result < 0 {
		return fail("set serial control lines failed (host error " + decimal(hostError(result)) + ")")
	}
	return nil
}

func (port *serial) write(data []byte) error {
	offset := 0
	wouldBlock := 0
	for offset < len(data) {
		n := hostWrite(port.fd, hostPointer(&data[offset]), len(data)-offset)
		if n < 0 && hostError(n) == hostWouldBlock() {
			wouldBlock++
			if wouldBlock >= 10000 {
				return fail("serial write remained blocked")
			}
			sleep(1)
			continue
		}
		if n <= 0 {
			return fail("serial write failed (host error " + decimal(hostError(n)) + ")")
		}
		wouldBlock = 0
		offset += n
	}
	return nil
}

func (port *serial) read(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := hostRead(port.fd, hostPointer(&data[0]), len(data))
	if n < 0 && hostError(n) == hostWouldBlock() {
		return 0
	}
	return n
}

type loader struct {
	port         *serial
	frame        []byte
	buffer       []byte
	bufferOffset int
	bufferLength int
	request      []byte
	escaped      bool
	inside       bool
}

func (loader *loader) identify() (target, []byte, error) {
	security, securityErr := loader.commandData(0x14, nil, 0, 1000)
	if securityErr == nil && len(security) >= 20 {
		chipID := u32(security, 12)
		for i := 0; i < len(targets); i++ {
			if uint32(targets[i].chipID) == chipID {
				copyOfSecurity := append([]byte{}, security...)
				return targets[i], copyOfSecurity, nil
			}
		}
		return target{}, nil, fail("unsupported connected ESP chip ID " + decimal(int(chipID)))
	}
	magic, err := loader.command(0x0a, words([]uint32{0x40001000}), 0, 1000)
	if err != nil {
		if securityErr != nil {
			return target{}, nil, fail("could not identify connected ESP chip: security info: " + securityErr.Error() + "; magic: " + err.Error())
		}
		return target{}, nil, fail("could not identify connected ESP chip: " + err.Error())
	}
	for i := 0; i < len(targets); i++ {
		if targets[i].magic != 0 && targets[i].magic == magic {
			return targets[i], nil, nil
		}
	}
	return target{}, nil, fail("unsupported connected ESP chip magic 0x" + hex(magic))
}

func (loader *loader) connect() error {
	var last error
	print("Entering ROM download mode\n")
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		if attempt == 0 {
			err = loader.usbReset()
		} else if attempt == 1 {
			err = loader.tightReset(50)
		} else {
			err = loader.tightReset(550)
		}
		if err == nil {
			loader.clearInput()
			if err = loader.sync(); err == nil {
				print("Synchronized\n")
				return nil
			}
		}
		last = err
		// Internal USB can disappear midway through a reset. Retry the sync
		// using the re-enumerated endpoint before issuing another reset.
		if err = loader.port.reopen(30); err != nil {
			last = err
			continue
		}
		loader.clearInput()
		if err = loader.sync(); err == nil {
			print("Synchronized\n")
			return nil
		}
		last = err
	}
	return fail("could not synchronize with ESP ROM loader: " + last.Error())
}

func (loader *loader) clearInput() {
	loader.frame = nil
	loader.bufferOffset = 0
	loader.bufferLength = 0
	loader.inside = false
	loader.escaped = false
}

func (loader *loader) usbReset() error {
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	sleep(100)
	if err := loader.port.setDTR(true); err != nil {
		return err
	}
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	sleep(100)
	if err := loader.port.setRTS(true); err != nil {
		return err
	}
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	if err := loader.port.setRTS(true); err != nil {
		return err
	}
	sleep(100)
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	return loader.port.setRTS(false)
}

func (loader *loader) classicReset(delay int) error {
	if err := loader.port.setDTR(false); err != nil {
		return err
	}
	if err := loader.port.setRTS(true); err != nil {
		return err
	}
	sleep(100)
	if err := loader.port.setDTR(true); err != nil {
		return err
	}
	if err := loader.port.setRTS(false); err != nil {
		return err
	}
	sleep(delay)
	return loader.port.setDTR(false)
}

func (loader *loader) tightReset(delay int) error {
	// Espressif's POSIX reset sequence uses TIOCMSET for each complete
	// DTR/RTS state. This avoids transient states between separate ioctls.
	if err := loader.port.setLines(false, false); err != nil {
		return err
	}
	if err := loader.port.setLines(true, true); err != nil {
		return err
	}
	if err := loader.port.setLines(false, true); err != nil {
		return err
	}
	sleep(100)
	if err := loader.port.setLines(true, false); err != nil {
		return err
	}
	sleep(delay)
	if err := loader.port.setLines(false, false); err != nil {
		return err
	}
	return loader.port.setDTR(false)
}

func (loader *loader) sync() error {
	payload := make([]byte, 36)
	payload[0], payload[1], payload[2], payload[3] = 7, 7, 0x12, 0x20
	for i := 4; i < len(payload); i++ {
		payload[i] = 0x55
	}
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := loader.command(0x08, payload, 0, 300); err == nil {
			return loader.drainSyncReplies()
		} else {
			last = err
		}
		sleep(80)
	}
	return last
}

func (loader *loader) drainSyncReplies() error {
	received := 0
	for empty := 0; received < 7 && empty < 100; {
		frame, hadData, err := loader.nextFrame()
		if err != nil {
			return err
		}
		if !hadData {
			empty++
			sleep(1)
			continue
		}
		if len(frame) >= 2 && frame[0] == 1 && frame[1] == 0x08 {
			received++
		}
	}
	if received != 7 {
		return fail("ESP SYNC response sequence was incomplete")
	}
	return nil
}

func (loader *loader) command(opcode byte, payload []byte, checksum uint32, emptyLimit int) (uint32, error) {
	value, _, err := loader.commandResponse(opcode, payload, checksum, emptyLimit)
	return value, err
}

func (loader *loader) commandData(opcode byte, payload []byte, checksum uint32, emptyLimit int) ([]byte, error) {
	_, body, err := loader.commandResponse(opcode, payload, checksum, emptyLimit)
	return body, err
}

func (loader *loader) commandResponse(opcode byte, payload []byte, checksum uint32, emptyLimit int) (uint32, []byte, error) {
	if err := loader.sendCommand(opcode, payload, checksum); err != nil {
		return 0, nil, err
	}
	for empty := 0; empty < emptyLimit; {
		frame, hadData, err := loader.nextFrame()
		if err != nil {
			return 0, nil, err
		}
		if !hadData {
			empty++
			sleep(1)
			continue
		}
		if len(frame) == 0 || len(frame) < 8 || frame[0] != 1 || frame[1] != opcode {
			continue
		}
		body := frame[8:]
		if len(body) >= 2 && body[len(body)-2] != 0 {
			return 0, nil, fail("ESP command 0x" + hex(uint32(opcode)) + " failed")
		}
		if len(body) >= 2 {
			body = body[:len(body)-2]
		}
		return u32(frame, 4), body, nil
	}
	return 0, nil, fail("ESP command 0x" + hex(uint32(opcode)) + " timed out")
}

func (loader *loader) sendCommand(opcode byte, payload []byte, checksum uint32) error {
	loader.request = loader.request[:0]
	loader.request = append(loader.request, 0xc0)
	header := []byte{0, opcode, byte(len(payload)), byte(len(payload) >> 8), byte(checksum), byte(checksum >> 8), byte(checksum >> 16), byte(checksum >> 24)}
	loader.request = appendEscaped(loader.request, header)
	loader.request = appendEscaped(loader.request, payload)
	loader.request = append(loader.request, 0xc0)
	if err := loader.port.write(loader.request); err != nil {
		return err
	}
	return nil
}

func (loader *loader) receiveFlashBatch(count int, output []byte) error {
	if len(loader.buffer) == 0 {
		loader.buffer = make([]byte, 256)
	}
	loader.frame = loader.frame[:0]
	escaped := false
	inside := false
	received := 0
	empty := 0
	for received < count {
		n := loader.port.read(loader.buffer)
		if n < 0 {
			return fail("serial read failed")
		}
		if n == 0 {
			empty++
			if empty > 10000 {
				return fail("ESP command 0xe timed out")
			}
			sleep(1)
			continue
		}
		empty = 0
		for i := 0; i < n; i++ {
			value := loader.buffer[i]
			if value == 0xc0 {
				if inside && len(loader.frame) != 0 {
					if len(loader.frame) < 8+backupChunk+2 || loader.frame[0] != 1 || loader.frame[1] != 0x0e {
						return fail("invalid pipelined flash response")
					}
					body := loader.frame[8:]
					if body[len(body)-2] != 0 {
						return fail("ESP command 0xe failed")
					}
					copy(output[received*backupChunk:(received+1)*backupChunk], body[:backupChunk])
					received++
					loader.frame = loader.frame[:0]
					escaped = false
					if received == count {
						return nil
					}
				}
				inside = true
				loader.frame = loader.frame[:0]
				escaped = false
			} else if !inside {
				continue
			} else if escaped {
				if value == 0xdc {
					loader.frame = append(loader.frame, 0xc0)
				} else if value == 0xdd {
					loader.frame = append(loader.frame, 0xdb)
				} else {
					loader.frame = append(loader.frame, 0xdb, value)
				}
				escaped = false
			} else if value == 0xdb {
				escaped = true
			} else {
				loader.frame = append(loader.frame, value)
			}
		}
	}
	return nil
}

func (loader *loader) nextFrame() ([]byte, bool, error) {
	if len(loader.buffer) == 0 {
		loader.buffer = make([]byte, 256)
	}
	hadData := false
	for {
		if loader.bufferOffset == loader.bufferLength {
			n := loader.port.read(loader.buffer)
			if n < 0 {
				return nil, false, fail("serial read failed")
			}
			if n == 0 {
				return nil, hadData, nil
			}
			loader.bufferOffset = 0
			loader.bufferLength = n
			hadData = true
		}
		value := loader.buffer[loader.bufferOffset]
		loader.bufferOffset++
		if value == 0xc0 {
			if loader.inside && len(loader.frame) != 0 {
				result := loader.frame
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
}

func slip(data []byte) []byte {
	result := []byte{0xc0}
	result = appendEscaped(result, data)
	return append(result, 0xc0)
}

func appendEscaped(result []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		if data[i] == 0xc0 {
			result = append(result, 0xdb, 0xdc)
		} else if data[i] == 0xdb {
			result = append(result, 0xdb, 0xdd)
		} else {
			result = append(result, data[i])
		}
	}
	return result
}

func readFile(path string) ([]byte, error) {
	fd := hostOpen(path, hostReadFlags(), 0)
	if fd < 0 {
		return nil, fail("read " + path + " failed")
	}
	defer hostClose(fd)
	var result []byte
	buffer := make([]byte, 4096)
	for {
		n := hostRead(fd, hostPointer(&buffer[0]), len(buffer))
		if n < 0 {
			return nil, fail("read " + path + " failed")
		}
		if n == 0 {
			return result, nil
		}
		result = append(result, buffer[:n]...)
	}
}

func writeFile(path string, data []byte) error {
	fd := hostOpen(path, hostCreateFlags(), 0x1b6)
	if fd < 0 {
		return fail("write " + path + " failed")
	}
	offset := 0
	for offset < len(data) {
		n := hostWrite(fd, hostPointer(&data[offset]), len(data)-offset)
		if n <= 0 {
			hostClose(fd)
			return fail("write " + path + " failed")
		}
		offset += n
	}
	hostClose(fd)
	return nil
}

func hostWriteAll(fd int, data []byte) int {
	written := 0
	for written < len(data) {
		n := hostWrite(fd, hostPointer(&data[written]), len(data)-written)
		if n <= 0 {
			return -1
		}
		written += n
	}
	return written
}

func u16(data []byte, at int) uint16 { return uint16(data[at]) | uint16(data[at+1])<<8 }
func u32(data []byte, at int) uint32 {
	return uint32(data[at]) | uint32(data[at+1])<<8 | uint32(data[at+2])<<16 | uint32(data[at+3])<<24
}
func word(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}
func words(values []uint32) []byte {
	var out []byte
	for i := 0; i < len(values); i++ {
		out = append(out, word(values[i])...)
	}
	return out
}
func put32(data []byte, at int, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
	data[at+2] = byte(value >> 16)
	data[at+3] = byte(value >> 24)
}
func put64(data []byte, at int, value int64) {
	for i := 0; i < 8; i++ {
		data[at+i] = byte(value >> uint(i*8))
	}
}
func bitCount(value byte) int {
	count := 0
	for value != 0 {
		count += int(value & 1)
		value >>= 1
	}
	return count
}
func decimal(value int) string {
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
func hex(value uint32) string {
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
