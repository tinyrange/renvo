//go:build linux && (amd64 || renvo)

package espflash

import (
	"os"
)

const (
	espressifVID = "303a"
	espJTAGPID   = "1001"

	usbJTAGInterface = 2
	usbJTAGOut       = 0x02
	usbJTAGIn        = 0x83

	usbdevfsBulk             = 0xc0185502
	usbdevfsClaimInterface   = 0x8004550f
	usbdevfsReleaseInterface = 0x80045510

	jtagIRIDCode = 0x01
	jtagIRDTMCS  = 0x10
	jtagIRDMI    = 0x11

	dmiData0      = 0x04
	dmiDMControl  = 0x10
	dmiDMStatus   = 0x11
	dmiAbstractCS = 0x16
	dmiCommand    = 0x17
	dmiProgram0   = 0x20
	dmiProgram1   = 0x21
	dmiSBCS       = 0x38
	dmiSBAddress0 = 0x39
	dmiSBData0    = 0x3c
)

// USBJTAG is a direct Linux usbfs client for the ESP32-C6 built-in
// USB-Serial/JTAG interface. It does not spawn OpenOCD or require libusb.
type USBJTAG struct {
	fd         int
	path       string
	abits      int
	idleClocks int
}

// DebugState is the selected ESP32-C6 hart state reported by the RISC-V debug
// module. Raw retains dmstatus for diagnostics and forward compatibility.
type DebugState struct {
	Raw         uint32
	Halted      bool
	Running     bool
	Unavailable bool
}

// OpenESP32C6JTAG claims the vendor JTAG interface and verifies both the TAP
// ID and RISC-V debug transport. An empty path auto-detects one 303a:1001 USB
// device.
func OpenESP32C6JTAG(path string) (*USBJTAG, error) {
	if path == "" {
		var err error
		path, err = detectUSBJTAG()
		if err != nil {
			return nil, err
		}
	}
	fd := hostOpen(path, 2, 0)
	if fd < 0 {
		return nil, fail("open USB/JTAG device " + path + " failed (host error " + decimal(hostError(fd)) + ")")
	}
	debug := &USBJTAG{fd: fd, path: path}
	interfaceNumber := make([]byte, 4)
	put32(interfaceNumber, 0, usbJTAGInterface)
	result := hostIoctl(fd, usbdevfsClaimInterface, hostPointer(&interfaceNumber[0]))
	if result < 0 {
		debug.Close()
		return nil, fail("claim ESP USB/JTAG interface failed (host error " + decimal(hostError(result)) + ")")
	}
	debug.drainInput()
	if err := debug.initialize(); err != nil {
		debug.Close()
		return nil, err
	}
	return debug, nil
}

func (debug *USBJTAG) Close() error {
	if debug == nil || debug.fd < 0 {
		return nil
	}
	interfaceNumber := make([]byte, 4)
	put32(interfaceNumber, 0, usbJTAGInterface)
	hostIoctl(debug.fd, usbdevfsReleaseInterface, hostPointer(&interfaceNumber[0]))
	result := hostClose(debug.fd)
	debug.fd = -1
	if result < 0 {
		return fail("close USB/JTAG device failed")
	}
	return nil
}

func (debug *USBJTAG) initialize() error {
	commands := newJTAGCommands()
	for i := 0; i < 6; i++ {
		commands.clock(0, 0, 1)
	}
	commands.clock(0, 0, 0)
	if _, err := debug.execute(commands); err != nil {
		return err
	}
	id, err := debug.scanDR(0, 32, 32)
	if err != nil {
		return err
	}
	if uint32(id) != 0x0000dc25 {
		return fail("JTAG TAP ID 0x" + hex(uint32(id)) + " is not ESP32-C6")
	}
	if _, err = debug.scanIR(jtagIRDTMCS); err != nil {
		return err
	}
	dtmcs, err := debug.scanDR(0, 32, 32)
	if err != nil {
		return err
	}
	if dtmcs&15 != 1 {
		return fail("unsupported RISC-V debug transport version " + decimal(int(dtmcs&15)))
	}
	debug.abits = int((dtmcs >> 4) & 0x3f)
	debug.idleClocks = int((dtmcs >> 12) & 7)
	if debug.abits < 6 || debug.abits > 16 {
		return fail("invalid RISC-V DMI address width " + decimal(debug.abits))
	}
	if _, err = debug.scanIR(jtagIRDTMCS); err != nil {
		return err
	}
	if _, err = debug.scanDR(1<<16, 32, 0); err != nil {
		return err
	}
	if _, err = debug.scanIR(jtagIRDMI); err != nil {
		return err
	}
	_, err = debug.dmiRead(dmiDMStatus)
	return err
}

func (debug *USBJTAG) Halt() error {
	// A freshly enumerated C6 commonly has allhavereset/anyhavereset latched.
	// Acknowledge it together with haltreq so the hart accepts the request.
	if err := debug.dmiWrite(dmiDMControl, 0x90000001); err != nil {
		return fail("halt ESP32-C6: " + err.Error())
	}
	lastStatus := uint32(0)
	for attempt := 0; attempt < 100; attempt++ {
		status, err := debug.dmiRead(dmiDMStatus)
		if err != nil {
			return err
		}
		lastStatus = status
		if status&(1<<9) != 0 {
			return nil
		}
		sleep(1)
	}
	control, _ := debug.dmiRead(dmiDMControl)
	return fail("ESP32-C6 did not halt (dmcontrol 0x" + hex(control) + ", dmstatus 0x" + hex(lastStatus) + ")")
}

// State reads the current hart state without changing it.
func (debug *USBJTAG) State() (DebugState, error) {
	var state DebugState
	status, err := debug.dmiRead(dmiDMStatus)
	if err != nil {
		return state, err
	}
	state.Raw = status
	state.Halted = status&(1<<9) != 0
	state.Running = status&(1<<11) != 0
	state.Unavailable = status&(1<<13) != 0
	return state, nil
}

func (debug *USBJTAG) Resume() error {
	if err := debug.dmiWrite(dmiDMControl, 0x40000001); err != nil {
		return fail("resume ESP32-C6: " + err.Error())
	}
	for attempt := 0; attempt < 100; attempt++ {
		status, err := debug.dmiRead(dmiDMStatus)
		if err != nil {
			return err
		}
		if status&(1<<17) != 0 {
			return debug.dmiWrite(dmiDMControl, 1)
		}
		sleep(1)
	}
	return fail("ESP32-C6 did not resume")
}

func (debug *USBJTAG) SetPC(address uint32) error {
	return debug.writeAbstractRegister(0x7b1, address)
}

// ReadPC reads the debug program counter. The hart must be halted.
func (debug *USBJTAG) ReadPC() (uint32, error) {
	return debug.readAbstractRegister(0x7b1)
}

// ReadRegister reads one integer register x0 through x31. The hart must be
// halted.
func (debug *USBJTAG) ReadRegister(register int) (uint32, error) {
	if register < 0 || register > 31 {
		return 0, fail("RISC-V register index must be between 0 and 31")
	}
	return debug.readAbstractRegister(uint32(0x1000 + register))
}

// WriteRegister writes one integer register x0 through x31. The hart must be
// halted.
func (debug *USBJTAG) WriteRegister(register int, value uint32) error {
	if register < 0 || register > 31 {
		return fail("RISC-V register index must be between 0 and 31")
	}
	return debug.writeAbstractRegister(uint32(0x1000+register), value)
}

// ReadRegisters reads all 32 integer registers from a halted hart.
func (debug *USBJTAG) ReadRegisters() ([]uint32, error) {
	registers := make([]uint32, 32)
	for register := 0; register < len(registers); register++ {
		value, err := debug.ReadRegister(register)
		if err != nil {
			return nil, err
		}
		registers[register] = value
	}
	return registers, nil
}

func (debug *USBJTAG) readAbstractRegister(register uint32) (uint32, error) {
	// Access Register: aarsize=32, transfer, read.
	if err := debug.dmiWrite(dmiCommand, 0x00220000|register); err != nil {
		return 0, err
	}
	if err := debug.waitAbstractCommand(); err != nil {
		return 0, err
	}
	return debug.dmiRead(dmiData0)
}

func (debug *USBJTAG) writeAbstractRegister(register uint32, value uint32) error {
	if err := debug.dmiWrite(dmiData0, value); err != nil {
		return err
	}
	// Access Register: aarsize=32, transfer, write.
	if err := debug.dmiWrite(dmiCommand, 0x00230000|register); err != nil {
		return err
	}
	return debug.waitAbstractCommand()
}

// Step executes one instruction and leaves the hart halted, returning its new
// debug program counter.
func (debug *USBJTAG) Step() (uint32, error) {
	state, err := debug.State()
	if err != nil {
		return 0, err
	}
	if !state.Halted {
		if err = debug.Halt(); err != nil {
			return 0, err
		}
	}
	dcsr, err := debug.readAbstractRegister(0x7b0)
	if err != nil {
		return 0, err
	}
	if err = debug.writeAbstractRegister(0x7b0, dcsr|(1<<2)); err != nil {
		return 0, err
	}
	if err = debug.dmiWrite(dmiDMControl, 0x40000001); err != nil {
		return 0, err
	}
	resumed := false
	for attempt := 0; attempt < 100; attempt++ {
		state, err = debug.State()
		if err != nil {
			return 0, err
		}
		if state.Raw&(1<<17) != 0 || state.Running {
			resumed = true
		}
		if resumed && state.Halted {
			break
		}
		sleep(1)
	}
	if err = debug.dmiWrite(dmiDMControl, 1); err != nil {
		return 0, err
	}
	if !resumed {
		return 0, fail("ESP32-C6 did not acknowledge single-step resume")
	}
	for attempt := 0; attempt < 100; attempt++ {
		state, err = debug.State()
		if err != nil {
			return 0, err
		}
		if state.Halted {
			if err = debug.writeAbstractRegister(0x7b0, dcsr&^(1<<2)); err != nil {
				return 0, err
			}
			return debug.ReadPC()
		}
		sleep(1)
	}
	return 0, fail("ESP32-C6 did not halt after single step")
}

func (debug *USBJTAG) FenceI() error {
	// Execute fence.i; ebreak from the two-entry program buffer.
	if err := debug.dmiWrite(dmiProgram0, 0x0000100f); err != nil {
		return err
	}
	if err := debug.dmiWrite(dmiProgram1, 0x00100073); err != nil {
		return err
	}
	if err := debug.dmiWrite(dmiCommand, 0x00240000); err != nil {
		return err
	}
	return debug.waitAbstractCommand()
}

func (debug *USBJTAG) waitAbstractCommand() error {
	for attempt := 0; attempt < 100; attempt++ {
		state, err := debug.dmiRead(dmiAbstractCS)
		if err != nil {
			return err
		}
		if state&(1<<12) != 0 {
			continue
		}
		if commandError := (state >> 8) & 7; commandError != 0 {
			debug.dmiWrite(dmiAbstractCS, 7<<8)
			return fail("RISC-V abstract command failed (cmderr " + decimal(int(commandError)) + ")")
		}
		return nil
	}
	return fail("RISC-V abstract command timed out")
}

func (debug *USBJTAG) WriteMemory(address uint32, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if address&3 != 0 || len(data)&3 != 0 {
		return fail("JTAG memory writes must be word aligned")
	}
	if uint32(len(data)) > ^uint32(0)-address {
		return fail("JTAG memory write range overflows the address space")
	}
	const wordsPerBatch = 128
	for batchAt := 0; batchAt < len(data); batchAt += wordsPerBatch * 4 {
		batchEnd := batchAt + wordsPerBatch*4
		if batchEnd > len(data) {
			batchEnd = len(data)
		}
		requests := make([]dmiRequest, 0, 2+(batchEnd-batchAt)/4)
		// 32-bit system-bus access with address autoincrement.
		requests = append(requests, dmiRequest{address: dmiSBCS, data: 0x00050000, operation: 2})
		requests = append(requests, dmiRequest{address: dmiSBAddress0, data: address + uint32(batchAt), operation: 2})
		for offset := batchAt; offset < batchEnd; offset += 4 {
			requests = append(requests, dmiRequest{address: dmiSBData0, data: u32(data, offset), operation: 2})
		}
		if err := debug.dmiBatch(requests); err != nil {
			return fail("write ESP32-C6 memory at 0x" + hex(address+uint32(batchAt)) + ": " + err.Error())
		}
	}
	state, err := debug.dmiRead(dmiSBCS)
	if err != nil {
		return err
	}
	if state&(1<<22) != 0 || state&(7<<12) != 0 {
		debug.dmiWrite(dmiSBCS, (1<<22)|(7<<12))
		return fail("RISC-V system-bus memory write failed")
	}
	return nil
}

// ReadMemory reads a word-aligned memory range through RISC-V system-bus
// access. Unlike the hot loader, reads are not restricted to SRAM.
func (debug *USBJTAG) ReadMemory(address uint32, size int) ([]byte, error) {
	if size <= 0 || address&3 != 0 || size&3 != 0 {
		return nil, fail("JTAG memory reads require a positive word-aligned range")
	}
	if uint32(size) > ^uint32(0)-address {
		return nil, fail("JTAG memory read range overflows the address space")
	}
	result := make([]byte, size)
	// Clear sticky errors, select 32-bit accesses, and read on address writes.
	if err := debug.dmiWrite(dmiSBCS, 0x00547000); err != nil {
		return nil, err
	}
	for offset := 0; offset < size; offset += 4 {
		if err := debug.dmiWrite(dmiSBAddress0, address+uint32(offset)); err != nil {
			return nil, err
		}
		if err := debug.waitSystemBus(); err != nil {
			return nil, err
		}
		value, err := debug.dmiRead(dmiSBData0)
		if err != nil {
			return nil, err
		}
		put32(result, offset, int(value))
	}
	if err := debug.dmiWrite(dmiSBCS, 0x00040000); err != nil {
		return nil, err
	}
	return result, nil
}

func (debug *USBJTAG) waitSystemBus() error {
	for attempt := 0; attempt < 100; attempt++ {
		state, err := debug.dmiRead(dmiSBCS)
		if err != nil {
			return err
		}
		if state&(1<<21) != 0 {
			continue
		}
		if state&(1<<22) != 0 || state&(7<<12) != 0 {
			debug.dmiWrite(dmiSBCS, (1<<22)|(7<<12))
			return fail("RISC-V system-bus access failed")
		}
		return nil
	}
	return fail("RISC-V system-bus access timed out")
}

type dmiRequest struct {
	address   uint32
	data      uint32
	operation uint32
}

func (debug *USBJTAG) dmiWrite(address uint32, data uint32) error {
	return debug.dmiBatch([]dmiRequest{{address: address, data: data, operation: 2}})
}

func (debug *USBJTAG) dmiRead(address uint32) (uint32, error) {
	request := uint64(address)<<34 | 1
	var lastResponse uint64
	for attempt := 0; attempt < 100; attempt++ {
		if _, err := debug.scanIR(jtagIRDMI); err != nil {
			return 0, err
		}
		bits := 34 + debug.abits
		commands := newJTAGCommands()
		commands.scanDR(request, bits, bits)
		idleClocks := debug.idleClocks + attempt
		if idleClocks < 1 {
			idleClocks = 1
		}
		for idle := 0; idle < idleClocks; idle++ {
			commands.clock(0, 0, 0)
		}
		commands.scanDR(0, bits, bits)
		responseData, err := debug.execute(commands)
		if err != nil {
			return 0, err
		}
		response := capturedBits(responseData, bits, bits)
		lastResponse = response
		switch response & 3 {
		case 0:
			if idleClocks > debug.idleClocks {
				debug.idleClocks = idleClocks
			}
			return uint32(response >> 2), nil
		case 3:
			if err = debug.resetDMI(); err != nil {
				return 0, err
			}
			continue
		default:
			return 0, fail("RISC-V DMI read failed")
		}
	}
	return 0, fail("RISC-V DMI remained busy (last response data 0x" + hex(uint32(lastResponse>>2)) + ")")
}

func (debug *USBJTAG) resetDMI() error {
	if _, err := debug.scanIR(jtagIRDTMCS); err != nil {
		return err
	}
	if _, err := debug.scanDR(1<<16, 32, 0); err != nil {
		return err
	}
	_, err := debug.scanIR(jtagIRDMI)
	return err
}

func (debug *USBJTAG) dmiBatch(requests []dmiRequest) error {
	if len(requests) == 0 {
		return nil
	}
	for attempt := 0; attempt < 100; attempt++ {
		if _, err := debug.scanIR(jtagIRDMI); err != nil {
			return err
		}
		commands := newJTAGCommands()
		for i := 0; i <= len(requests); i++ {
			value := uint64(0)
			if i < len(requests) {
				value = uint64(requests[i].address)<<34 | uint64(requests[i].data)<<2 | uint64(requests[i].operation)
			}
			commands.scanDR(value, 34+debug.abits, 2)
			idleClocks := debug.idleClocks
			if idleClocks < 1 {
				idleClocks = 1
			}
			for idle := 0; idle < idleClocks; idle++ {
				commands.clock(0, 0, 0)
			}
		}
		response, err := debug.execute(commands)
		if err != nil {
			return err
		}
		busy := false
		for i := 1; i <= len(requests); i++ {
			status := capturedBits(response, i*2, 2)
			if status == 3 {
				busy = true
				break
			}
			if status != 0 {
				return fail("RISC-V DMI write failed")
			}
		}
		if !busy {
			return nil
		}
		if err = debug.resetDMI(); err != nil {
			return err
		}
		debug.idleClocks++
	}
	return fail("RISC-V DMI remained busy during batched write")
}

func (debug *USBJTAG) scanIR(value uint64) (uint64, error) {
	commands := newJTAGCommands()
	commands.clock(0, 0, 1)
	commands.clock(0, 0, 1)
	commands.clock(0, 0, 0)
	commands.clock(0, 0, 0)
	commands.shift(value, 5, 5)
	commands.clock(0, 0, 1)
	commands.clock(0, 0, 0)
	response, err := debug.execute(commands)
	return capturedBits(response, 0, 5), err
}

func (debug *USBJTAG) scanDR(value uint64, bits int, capture int) (uint64, error) {
	commands := newJTAGCommands()
	commands.scanDR(value, bits, capture)
	response, err := debug.execute(commands)
	return capturedBits(response, 0, capture), err
}

type jtagCommands struct {
	data     []byte
	nibbles  int
	captures int
}

func newJTAGCommands() *jtagCommands { return &jtagCommands{} }

func (commands *jtagCommands) nibble(value byte) {
	if commands.nibbles&1 == 0 {
		commands.data = append(commands.data, value<<4)
	} else {
		commands.data[len(commands.data)-1] |= value & 15
	}
	commands.nibbles++
}

func (commands *jtagCommands) clock(capture int, tdi int, tms int) {
	commands.nibble(byte((capture&1)<<2 | (tms&1)<<1 | (tdi & 1)))
	if capture != 0 {
		commands.captures++
	}
}

func (commands *jtagCommands) shift(value uint64, bits int, capture int) {
	for bit := 0; bit < bits; bit++ {
		last := 0
		if bit+1 == bits {
			last = 1
		}
		read := 0
		if bit < capture {
			read = 1
		}
		commands.clock(read, int((value>>bit)&1), last)
	}
}

func (commands *jtagCommands) scanDR(value uint64, bits int, capture int) {
	commands.clock(0, 0, 1)
	commands.clock(0, 0, 0)
	commands.clock(0, 0, 0)
	commands.shift(value, bits, capture)
	commands.clock(0, 0, 1)
	commands.clock(0, 0, 0)
}

func (debug *USBJTAG) execute(commands *jtagCommands) ([]byte, error) {
	if debug == nil || debug.fd < 0 {
		return nil, fail("USB/JTAG device is closed")
	}
	commands.nibble(0x0a)
	if commands.nibbles&1 != 0 {
		commands.nibble(0x0a)
	}
	if err := debug.bulkWrite(commands.data); err != nil {
		return nil, err
	}
	bytesExpected := (commands.captures + 7) / 8
	response := make([]byte, bytesExpected)
	offset := 0
	empty := 0
	for offset < len(response) {
		read, err := debug.bulkTransfer(usbJTAGIn, response[offset:])
		if err != nil {
			return nil, err
		}
		if read <= 0 {
			empty++
			if empty >= 100 {
				return nil, fail("USB/JTAG returned an empty response")
			}
			sleep(1)
			continue
		}
		empty = 0
		offset += read
	}
	return response, nil
}

func (debug *USBJTAG) bulkWrite(data []byte) error {
	offset := 0
	for offset < len(data) {
		written, err := debug.bulkTransfer(usbJTAGOut, data[offset:])
		if err != nil {
			return err
		}
		if written <= 0 {
			return fail("USB/JTAG write made no progress")
		}
		offset += written
	}
	return nil
}

func (debug *USBJTAG) bulkTransfer(endpoint int, data []byte) (int, error) {
	return debug.bulkTransferTimeout(endpoint, data, 1000)
}

func (debug *USBJTAG) bulkTransferTimeout(endpoint int, data []byte, timeout int) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	transfer := make([]byte, 24)
	put32(transfer, 0, endpoint)
	put32(transfer, 4, len(data))
	put32(transfer, 8, timeout)
	put64(transfer, 16, int64(hostPointer(&data[0])))
	result := hostIoctl(debug.fd, usbdevfsBulk, hostPointer(&transfer[0]))
	if result < 0 {
		return 0, fail("USB/JTAG endpoint 0x" + hex(uint32(endpoint)) + " transfer failed (host error " + decimal(hostError(result)) + ")")
	}
	return result, nil
}

func (debug *USBJTAG) drainInput() {
	buffer := make([]byte, 64)
	for attempt := 0; attempt < 16; attempt++ {
		read, err := debug.bulkTransferTimeout(usbJTAGIn, buffer, 1)
		if err != nil || read <= 0 {
			return
		}
	}
}

func capturedBits(data []byte, offset int, count int) uint64 {
	var result uint64
	for bit := 0; bit < count; bit++ {
		at := offset + bit
		if at/8 < len(data) && data[at/8]&(1<<uint(at&7)) != 0 {
			result |= 1 << uint(bit)
		}
	}
	return result
}

func detectUSBJTAG() (string, error) {
	entries, err := os.ReadDir("/sys/bus/usb/devices")
	if err != nil {
		return "", fail("could not scan sysfs for ESP USB/JTAG")
	}
	var matches []string
	for i := 0; i < len(entries); i++ {
		base := "/sys/bus/usb/devices/" + entries[i].Name()
		vendor, vendorErr := readTrimmed(base + "/idVendor")
		product, productErr := readTrimmed(base + "/idProduct")
		if vendorErr != nil || productErr != nil || vendor != espressifVID || product != espJTAGPID {
			continue
		}
		bus, busErr := readTrimmed(base + "/busnum")
		device, deviceErr := readTrimmed(base + "/devnum")
		if busErr != nil || deviceErr != nil {
			continue
		}
		matches = append(matches, "/dev/bus/usb/"+padUSBNumber(bus)+"/"+padUSBNumber(device))
	}
	if len(matches) == 0 {
		return "", fail("no Espressif 303a:1001 USB/JTAG device found")
	}
	if len(matches) != 1 {
		return "", fail("multiple Espressif USB/JTAG devices found; pass the device path explicitly")
	}
	return matches[0], nil
}

func readTrimmed(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	start := 0
	end := len(data)
	for start < end && (data[start] == ' ' || data[start] == '\n' || data[start] == '\r' || data[start] == '\t') {
		start++
	}
	for end > start && (data[end-1] == ' ' || data[end-1] == '\n' || data[end-1] == '\r' || data[end-1] == '\t') {
		end--
	}
	return string(data[start:end]), nil
}

func padUSBNumber(value string) string {
	for len(value) < 3 {
		value = "0" + value
	}
	return value
}
