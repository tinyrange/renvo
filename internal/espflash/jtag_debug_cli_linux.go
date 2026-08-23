//go:build linux && (amd64 || renvo)

package espflash

func isJTAGDebugMode(mode string) bool {
	return mode == "--jtag-status" || mode == "--jtag-halt" ||
		mode == "--jtag-resume" || mode == "--jtag-regs" ||
		mode == "--jtag-step" || mode == "--jtag-read" ||
		mode == "--jtag-set-pc" || mode == "--jtag-set-reg" ||
		mode == "--jtag-blink"
}

func runJTAGDebugCommand(args []string) int {
	mode := args[1]
	values := 0
	if mode == "--jtag-read" || mode == "--jtag-set-reg" || mode == "--jtag-blink" {
		values = 2
	} else if mode == "--jtag-set-pc" {
		values = 1
	}
	if len(args) < 2+values || len(args) > 3+values {
		printJTAGDebugUsage(mode)
		return 2
	}
	path := ""
	if len(args) == 3+values {
		path = args[len(args)-1]
	}
	debug, err := OpenESP32C6JTAG(path)
	if err != nil {
		print("renvoflash: " + err.Error() + "\n")
		return 1
	}
	defer debug.Close()
	if mode == "--jtag-status" {
		return printJTAGStatus(debug)
	}
	if mode == "--jtag-halt" {
		if err = debug.Halt(); err == nil {
			var pc uint32
			pc, err = debug.ReadPC()
			if err == nil {
				print("ESP32-C6 halted at pc=0x" + hex32(pc) + "\n")
			}
		}
		return printJTAGError(err)
	}
	if mode == "--jtag-resume" {
		err = debug.Resume()
		if err == nil {
			print("ESP32-C6 resumed\n")
		}
		return printJTAGError(err)
	}
	if mode == "--jtag-step" {
		var pc uint32
		pc, err = debug.Step()
		if err == nil {
			print("ESP32-C6 stepped to pc=0x" + hex32(pc) + " and is halted\n")
		}
		return printJTAGError(err)
	}
	if mode == "--jtag-regs" {
		return printJTAGRegisters(debug)
	}
	if mode == "--jtag-read" {
		return readJTAGMemory(debug, args[2], args[3])
	}
	if mode == "--jtag-set-pc" {
		return setJTAGPC(debug, args[2])
	}
	if mode == "--jtag-blink" {
		return blinkJTAGLED(debug, args[2], args[3])
	}
	return setJTAGRegister(debug, args[2], args[3])
}

func printJTAGDebugUsage(mode string) {
	if mode == "--jtag-read" {
		print("usage: renvoflash --jtag-read ADDRESS SIZE [USB-DEVICE]\n")
	} else if mode == "--jtag-blink" {
		print("usage: renvoflash --jtag-blink CYCLES HALF-PERIOD-MS [USB-DEVICE]\n")
	} else if mode == "--jtag-set-pc" {
		print("usage: renvoflash --jtag-set-pc ADDRESS [USB-DEVICE]\n")
	} else if mode == "--jtag-set-reg" {
		print("usage: renvoflash --jtag-set-reg REGISTER VALUE [USB-DEVICE]\n")
	} else {
		print("usage: renvoflash " + mode + " [USB-DEVICE]\n")
	}
}

func writeJTAGWord(debug *USBJTAG, address uint32, value uint32) error {
	data := make([]byte, 4)
	put32(data, 0, int(value))
	return debug.WriteMemory(address, data)
}

// blinkJTAGLED drives the M5NanoC6's active-high GPIO7 LED while the hart is
// halted. All timing and every level transition therefore originate on the
// host, not in firmware already running on the microcontroller.
func blinkJTAGLED(debug *USBJTAG, cyclesText string, halfPeriodText string) int {
	cycles, ok := parseDebugUint32(cyclesText)
	if !ok || cycles == 0 || cycles > 1000 {
		return printJTAGError(fail("JTAG blink cycles must be between 1 and 1000"))
	}
	halfPeriod, ok := parseDebugUint32(halfPeriodText)
	if !ok || halfPeriod == 0 || halfPeriod > 60000 {
		return printJTAGError(fail("JTAG blink half-period must be between 1 and 60000 milliseconds"))
	}
	state, err := debug.State()
	if err != nil {
		return printJTAGError(err)
	}
	resume := state.Running
	if !state.Halted {
		err = debug.Halt()
	}
	const (
		gpio7IOMux        = uint32(0x60090020)
		gpio7OutputSelect = uint32(0x60091570)
		gpioOutSet        = uint32(0x60091008)
		gpioOutClear      = uint32(0x6009100c)
		gpioEnableSet     = uint32(0x60091024)
		gpio7             = uint32(1 << 7)
		gpioFunctionMask  = uint32(7 << 12)
		gpioInputEnable   = uint32(1 << 9)
		gpioPullUp        = uint32(1 << 8)
		gpioPullDown      = uint32(1 << 7)
	)
	if err == nil {
		var mux []byte
		mux, err = debug.ReadMemory(gpio7IOMux, 4)
		if err == nil {
			value := u32(mux, 0) &^ (gpioFunctionMask | gpioInputEnable | gpioPullUp | gpioPullDown)
			err = writeJTAGWord(debug, gpio7IOMux, value|1<<12)
		}
	}
	if err == nil {
		err = writeJTAGWord(debug, gpio7OutputSelect, 128)
	}
	if err == nil {
		err = writeJTAGWord(debug, gpioEnableSet, gpio7)
	}
	for cycle := uint32(0); err == nil && cycle < cycles; cycle++ {
		err = writeJTAGWord(debug, gpioOutSet, gpio7)
		if err == nil {
			sleep(int(halfPeriod))
			err = writeJTAGWord(debug, gpioOutClear, gpio7)
		}
		if err == nil && cycle+1 < cycles {
			sleep(int(halfPeriod))
		}
	}
	offErr := writeJTAGWord(debug, gpioOutClear, gpio7)
	if err == nil {
		err = offErr
	}
	if resume {
		resumeErr := debug.Resume()
		if err == nil {
			err = resumeErr
		}
	}
	if err != nil {
		return printJTAGError(err)
	}
	print("Blinked M5NanoC6 GPIO7 LED " + decimal(int(cycles)) + " times under host JTAG control\n")
	return 0
}

func printJTAGError(err error) int {
	if err != nil {
		print("renvoflash: " + err.Error() + "\n")
		return 1
	}
	return 0
}

func printJTAGStatus(debug *USBJTAG) int {
	state, err := debug.State()
	if err != nil {
		return printJTAGError(err)
	}
	name := "unknown"
	if state.Halted {
		name = "halted"
	} else if state.Running {
		name = "running"
	} else if state.Unavailable {
		name = "unavailable"
	}
	print("ESP32-C6 " + name + " (dmstatus=0x" + hex32(state.Raw) + ")\n")
	return 0
}

func printJTAGRegisters(debug *USBJTAG) int {
	state, err := debug.State()
	if err != nil {
		return printJTAGError(err)
	}
	resume := state.Running
	if !state.Halted {
		if err = debug.Halt(); err != nil {
			return printJTAGError(err)
		}
	}
	pc, err := debug.ReadPC()
	var registers []uint32
	if err == nil {
		registers, err = debug.ReadRegisters()
	}
	if resume {
		resumeErr := debug.Resume()
		if err == nil {
			err = resumeErr
		}
	}
	if err != nil {
		return printJTAGError(err)
	}
	print("pc       0x" + hex32(pc) + "\n")
	for register := 0; register < len(registers); register++ {
		print(registerLabel(register) + " 0x" + hex32(registers[register]) + "\n")
	}
	return 0
}

func readJTAGMemory(debug *USBJTAG, addressText string, sizeText string) int {
	address, ok := parseDebugUint32(addressText)
	if !ok {
		return printJTAGError(fail("invalid JTAG memory address " + addressText))
	}
	sizeValue, ok := parseDebugUint32(sizeText)
	if !ok || sizeValue == 0 || sizeValue > 4096 {
		return printJTAGError(fail("JTAG memory size must be between 1 and 4096 bytes"))
	}
	state, err := debug.State()
	if err != nil {
		return printJTAGError(err)
	}
	resume := state.Running
	if !state.Halted {
		err = debug.Halt()
	}
	var data []byte
	if err == nil {
		data, err = debug.ReadMemory(address, int(sizeValue))
	}
	if resume {
		resumeErr := debug.Resume()
		if err == nil {
			err = resumeErr
		}
	}
	if err != nil {
		return printJTAGError(err)
	}
	printMemoryDump(address, data)
	return 0
}

func setJTAGPC(debug *USBJTAG, valueText string) int {
	value, ok := parseDebugUint32(valueText)
	if !ok {
		return printJTAGError(fail("invalid program counter " + valueText))
	}
	if err := debug.Halt(); err != nil {
		return printJTAGError(err)
	}
	if err := debug.SetPC(value); err != nil {
		return printJTAGError(err)
	}
	print("pc set to 0x" + hex32(value) + "; ESP32-C6 remains halted\n")
	return 0
}

func setJTAGRegister(debug *USBJTAG, name string, valueText string) int {
	register, ok := parseRegister(name)
	if !ok {
		return printJTAGError(fail("invalid RISC-V register " + name))
	}
	value, ok := parseDebugUint32(valueText)
	if !ok {
		return printJTAGError(fail("invalid register value " + valueText))
	}
	if err := debug.Halt(); err != nil {
		return printJTAGError(err)
	}
	if err := debug.WriteRegister(register, value); err != nil {
		return printJTAGError(err)
	}
	print(registerName(register) + " set to 0x" + hex32(value) + "; ESP32-C6 remains halted\n")
	return 0
}

func parseDebugUint32(text string) (uint32, bool) {
	if text == "" {
		return 0, false
	}
	base := uint32(10)
	start := 0
	if len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		base = 16
		start = 2
	}
	if start == len(text) {
		return 0, false
	}
	value := uint32(0)
	for i := start; i < len(text); i++ {
		digit := uint32(255)
		if text[i] >= '0' && text[i] <= '9' {
			digit = uint32(text[i] - '0')
		} else if base == 16 && text[i] >= 'a' && text[i] <= 'f' {
			digit = uint32(text[i]-'a') + 10
		} else if base == 16 && text[i] >= 'A' && text[i] <= 'F' {
			digit = uint32(text[i]-'A') + 10
		}
		if digit >= base || value > (^uint32(0)-digit)/base {
			return 0, false
		}
		value = value*base + digit
	}
	return value, true
}

func parseRegister(name string) (int, bool) {
	if len(name) >= 2 && name[0] == 'x' {
		value, ok := parseDebugUint32(name[1:])
		if ok && value < 32 {
			return int(value), true
		}
	}
	for register := 0; register < 32; register++ {
		if name == registerName(register) {
			return register, true
		}
	}
	return 0, false
}

func registerName(register int) string {
	names := []string{
		"zero", "ra", "sp", "gp", "tp", "t0", "t1", "t2",
		"s0", "s1", "a0", "a1", "a2", "a3", "a4", "a5",
		"a6", "a7", "s2", "s3", "s4", "s5", "s6", "s7",
		"s8", "s9", "s10", "s11", "t3", "t4", "t5", "t6",
	}
	if register < 0 || register >= len(names) {
		return "?"
	}
	return names[register]
}

func registerLabel(register int) string {
	label := "x" + decimal(register) + "/" + registerName(register)
	for len(label) < 8 {
		label += " "
	}
	return label
}

func hex32(value uint32) string {
	result := hex(value)
	for len(result) < 8 {
		result = "0" + result
	}
	return result
}

func hexByte(value byte) string {
	digits := "0123456789abcdef"
	return string([]byte{digits[value>>4], digits[value&15]})
}

func printMemoryDump(address uint32, data []byte) {
	for offset := 0; offset < len(data); offset += 16 {
		end := offset + 16
		if end > len(data) {
			end = len(data)
		}
		line := hex32(address+uint32(offset)) + ":"
		for i := offset; i < end; i++ {
			line += " " + hexByte(data[i])
		}
		print(line + "\n")
	}
}
