//go:build m5tab5

package board

const (
	internalSDA  = touchSDA
	internalSCL  = touchSCL
	powerAddress = byte(0x43)
)

var powerInitialized bool
var panelReset bool
var powerFailure int

// PowerFailure reports which PI4IO setup transaction failed during bring-up.
func PowerFailure() int { return powerFailure }

// InitPower establishes the Tab5's board-control outputs. In particular it
// holds the panel and touch controller out of reset independent of state
// retained from a previous firmware. InitTouch disables the speaker after the
// controller has had the same release sequence as the factory application.
func InitPower() bool {
	if powerInitialized {
		return true
	}
	powerFailure = 0
	configureI2C(internalSDA, internalSCL)
	// Match the factory UserDemo BSP used by this ST7121 unit. Resetting the
	// PI4IO itself establishes a real low pulse on LCD_RST and TP_RST before
	// their configured output values release both controllers.
	resetOK := i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x01, 0xff)
	// On a cold USB-download boot the PI4IO begins resetting as soon as it
	// receives 0xff and can NACK that third and final byte. A warm JTAG reset
	// normally observes the ACK. Accept only this exact reset-value NACK; the
	// configuration writes below still prove that the expander restarted.
	resetValueNACK := lastI2CEvent&(1<<10) != 0 && lastI2CByte == 3
	if !resetOK && !resetValueNACK {
		powerFailure = 1
		return false
	}
	// The reset command ACKs before the expander has restarted its I2C state
	// machine. ESP-IDF's queued driver naturally leaves this small scheduling
	// gap; the bare-metal path must provide it explicitly.
	panelDelay(10)
	// The factory BSP issues a CHIP_RESET readback but deliberately ignores its
	// status. This unit NACKs that reset/write-only register, so preserve the
	// transaction and ordering without treating it as a functional failure.
	i2cReadRegister8(internalSDA, internalSCL, powerAddress, 0x01)
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x03, 0x7f) {
		powerFailure = 3
		return false
	}
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x07, 0x00) {
		powerFailure = 4
		return false
	}
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x0d, 0x7f) {
		powerFailure = 5
		return false
	}
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x0b, 0x7f) {
		powerFailure = 6
		return false
	}
	// Match the factory board state by releasing CAM_RST, LCD_RST and TP_RST.
	// Keep SPK_EN low so bring-up remains silent.
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x05, 0x74) {
		powerFailure = 7
		return false
	}
	powerInitialized = true
	return true
}

func muteSpeaker() bool {
	return i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x05, 0x74)
}

func resetPanelAndTouch() bool {
	if panelReset {
		return true
	}
	// Release TP_INT before pulsing the panel and touch reset outputs, matching
	// the current M5Stack UserDemo sequence for integrated ST712x panels.
	configureGPIO(23, true, false)
	enableGPIO(23, false)
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x05, 0x44) {
		return false
	}
	panelDelay(100)
	if !i2cWriteRegister8(internalSDA, internalSCL, powerAddress, 0x05, 0x74) {
		return false
	}
	panelDelay(100)
	panelReset = true
	return true
}

func verifyTouchReleased() bool {
	direction, directionOK := i2cReadRegister8(internalSDA, internalSCL, powerAddress, 0x03)
	output, outputOK := i2cReadRegister8(internalSDA, internalSCL, powerAddress, 0x05)
	highImpedance, highImpedanceOK := i2cReadRegister8(internalSDA, internalSCL, powerAddress, 0x07)
	return directionOK && outputOK && highImpedanceOK &&
		direction&0x20 != 0 && output&0x20 != 0 && highImpedance&0x20 == 0
}
