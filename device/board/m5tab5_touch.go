//go:build m5tab5

package board

import touchcontroller "renvo.dev/device/input/st7121"

const touchAddress = touchcontroller.Address

// TouchPoint is one filtered contact in the panel's native coordinate space.
type TouchPoint = touchcontroller.Point

// TouchReportStats describes the most recently decoded controller report.
type TouchReportStats = touchcontroller.ReportStats

var touchMaximum = 10
var touchFirmware int
var touchMiscellaneous int
var touchReadFailure int
var touchIRQBefore int
var touchIRQAfter int
var touchLastAdvanced int
var touchLastPoints [10]TouchPoint
var touchLastCount int
var touchLastRawPoints [10]TouchPoint
var touchLastRawCount int
var touchSensingCounter int
var touchChecksum = -1
var touchChecksumCoordSum int
var touchChecksumReportSum int
var touchChecksumCoordXOR int
var touchReports int
var tab5TouchFilter touchcontroller.Filter

func copyLastTouches(points []TouchPoint) int {
	count := touchLastCount
	if count > len(points) {
		count = len(points)
	}
	for index := 0; index < count; index++ {
		points[index] = touchLastPoints[index]
	}
	return count
}

// InitTouch verifies the ST7121 firmware and its native coordinate range.
func InitTouch() bool {
	if !InitPower() {
		return false
	}
	// Board-control setup runs at 100 kHz. The ST7121's normal polling rate is
	// 400 kHz, matching M5Stack's driver and cutting a full report to one quarter
	// of the bus time.
	if !configureTouchI2C() {
		print("TAB5 TOUCH I2C SPEED FAIL\n")
		return false
	}
	// Probe only the address before attempting a register transaction. This
	// distinguishes reset/power failures from pointer/read protocol failures.
	if !i2cWrite(touchAddress, nil) {
		print("TAB5 TOUCH ENDPOINT FAIL\n")
		return false
	}
	print("TAB5 TOUCH ENDPOINT PASS\n")
	var info [16]byte
	read := false
	for attempt := 0; attempt < 6 && !read; attempt++ {
		read = i2cReadRegister(touchSDA, touchSCL, touchAddress, 0, info[:])
		if !read {
			panelDelay(10)
		}
	}
	failure := lastI2CFailure
	startFailure := lastStartFailure
	event := lastI2CEvent
	failedByte := lastI2CByte
	state := lastI2CState
	if !read {
		// Failure diagnostics address the board expander, so restore its 100 kHz
		// timing first. InitPower already left the speaker output disabled on the
		// successful path; no expander transaction belongs in the touch loop.
		configureI2C(internalSDA, internalSCL)
		muteSpeaker()
		if verifyTouchReleased() {
			print("TAB5 TOUCH RESET RELEASED\n")
		} else {
			print("TAB5 TOUCH RESET HELD\n")
		}
		if failure == 1 {
			if startFailure == 1 {
				print("TAB5 TOUCH START SCL FAIL\n")
			} else {
				print("TAB5 TOUCH START SDA FAIL\n")
			}
		} else if failure == 5 {
			print("TAB5 TOUCH RESTART FAIL\n")
		} else if failure == 2 {
			print("TAB5 TOUCH WRITE ADDRESS FAIL\n")
		} else if failure == 3 || failure == 4 {
			print("TAB5 TOUCH REGISTER FAIL\n")
		} else if failure == 6 {
			print("TAB5 TOUCH READ ADDRESS FAIL\n")
		} else if failure == 10 {
			print("TAB5 TOUCH POINTER PHASE FAIL\n")
		} else if failure == 11 {
			print("TAB5 TOUCH ADDRESS PHASE FAIL\n")
		} else if failure == 12 {
			print("TAB5 TOUCH ACK READ PHASE FAIL\n")
		} else if failure == 13 {
			print("TAB5 TOUCH NACK READ PHASE FAIL\n")
		} else if failure == 14 {
			print("TAB5 TOUCH STOP PHASE FAIL\n")
		} else {
			print("TAB5 TOUCH DATA FAIL\n")
		}
		if event == 0xffffffff {
			print("TAB5 TOUCH I2C POLL TIMEOUT\n")
			if state&(1<<4) != 0 {
				print("TAB5 TOUCH I2C BUS BUSY\n")
			} else {
				print("TAB5 TOUCH I2C BUS IDLE\n")
			}
			mainState := state >> 24 & 7
			if mainState == 3 {
				print("TAB5 TOUCH I2C RECEIVE STATE\n")
			} else if mainState == 4 {
				print("TAB5 TOUCH I2C TRANSMIT STATE\n")
			} else if mainState == 5 {
				print("TAB5 TOUCH I2C SEND ACK STATE\n")
			} else if mainState == 6 {
				print("TAB5 TOUCH I2C WAIT ACK STATE\n")
			}
			if state>>8&0x3f == 0 {
				print("TAB5 TOUCH I2C RX FIFO EMPTY\n")
			} else {
				print("TAB5 TOUCH I2C RX FIFO DATA\n")
			}
		} else if event&(1<<10) != 0 {
			print("TAB5 TOUCH I2C NACK\n")
			if failedByte == 1 {
				print("TAB5 TOUCH I2C ADDRESS NACK\n")
			} else if failedByte == 2 {
				print("TAB5 TOUCH I2C REGISTER HIGH NACK\n")
			} else if failedByte == 3 {
				print("TAB5 TOUCH I2C REGISTER LOW NACK\n")
			}
		} else if event&(1<<8) != 0 {
			print("TAB5 TOUCH I2C TIMEOUT\n")
		} else if event&(1<<13) != 0 {
			print("TAB5 TOUCH I2C STATE TIMEOUT\n")
		} else if event&(1<<5) != 0 {
			print("TAB5 TOUCH I2C ARBITRATION FAIL\n")
		}
		return false
	}
	maximumX := int(info[5])<<8 | int(info[6])
	maximumY := int(info[7])<<8 | int(info[8])
	touchMaximum = int(info[9])
	touchFirmware = int(info[0])
	var miscellaneous [1]byte
	if !i2cReadRegister(touchSDA, touchSCL, touchAddress, 0x00f0, miscellaneous[:]) {
		print("TAB5 TOUCH MISC INFO FAIL\n")
		return false
	}
	touchMiscellaneous = int(miscellaneous[0])
	touchLastCount = 0
	touchLastRawCount = 0
	touchSensingCounter = 0
	touchChecksum = -1
	touchChecksumCoordSum = 0
	touchChecksumReportSum = 0
	touchChecksumCoordXOR = 0
	touchReports = 0
	tab5TouchFilter.Reset()
	// TP_INT pulses low for each sensing frame. Hardware edge latching retains
	// that event while the renderer is busy at a frame boundary.
	configureGPIOFallingEdge(23)
	if touchFirmware != 1 {
		print("TAB5 TOUCH FIRMWARE FAIL\n")
		return false
	}
	if maximumX != DisplayWidth || maximumY != DisplayHeight {
		print("TAB5 TOUCH GEOMETRY FAIL\n")
		return false
	}
	if touchMaximum <= 0 || touchMaximum > 10 {
		print("TAB5 TOUCH COUNT FAIL\n")
		return false
	}
	return true
}

// ReadTouches returns all valid simultaneous contacts reported by ST7121.
// The caller supplies storage so polling does not allocate.
func ReadTouches(points []TouchPoint) (int, bool) {
	touchReadFailure = 0
	touchIRQBefore = 0
	interruptHigh := readGPIO(23)
	if interruptHigh {
		touchIRQBefore = 1
	}
	ready := gpioInterruptPending(23) || !interruptHigh
	if !ready {
		touchIRQAfter = 1
		// Never read the report page without a real data-ready event. Such reads
		// produce coherent synthetic edge/grid contacts on the integrated ST7121.
		return copyLastTouches(points), true
	}
	// Clear before reading so a new falling edge during this transaction remains
	// latched for the next call instead of being erased after the report ACK.
	clearGPIOInterrupt(23)
	// Read Advanced Touch Info and all coordinate slots as one report-page
	// transaction. Register 0x000a's sensing counter cannot be included in this
	// read: although numerically adjacent, the ST7121 does not auto-increment
	// from that status region into the report page on this firmware. Reading
	// through the final coordinate byte acknowledges the returned snapshot.
	var page [74]byte
	pageLength := touchcontroller.ReportSize(touchMaximum)
	if !i2cReadRegister(touchSDA, touchSCL, touchAddress, 0x0010, page[:pageLength]) {
		touchReadFailure = 1
		return 0, false
	}
	var raw [touchcontroller.MaximumContacts]TouchPoint
	advanced, rawCount, decoded := touchcontroller.DecodeReport(
		page[:pageLength], touchMaximum, DisplayWidth, DisplayHeight, raw[:],
	)
	if !decoded {
		touchReadFailure = 2
		return 0, false
	}
	touchLastAdvanced = advanced
	touchReports++
	// Firmware 1.80.1.16 advertises checksum capability in Miscellaneous Info,
	// but its Advanced Touch Info value does not include a checksum in the active
	// report. Do not fetch CkAddr separately: it is not tied atomically to this
	// frame and cannot safely validate or reject coordinates.
	touchChecksum = -1
	touchChecksumCoordSum = 0
	touchChecksumReportSum = 0
	touchChecksumCoordXOR = 0
	if advanced&0x08 == 0 {
		touchLastRawCount = 0
		touchIRQAfter = 0
		if readGPIO(23) {
			touchIRQAfter = 1
		}
		return copyLastTouches(points), true
	}
	touchLastRawCount = rawCount
	for index := 0; index < rawCount; index++ {
		touchLastRawPoints[index] = raw[index]
	}
	count := tab5TouchFilter.Apply(raw[:rawCount], points)
	touchIRQAfter = 0
	if readGPIO(23) {
		touchIRQAfter = 1
	}
	touchLastCount = count
	for index := 0; index < count; index++ {
		touchLastPoints[index] = points[index]
	}
	return count, true
}

// TouchProtocolInfo reports the controller identity and optional feature bits.
func TouchProtocolInfo() (int, int) {
	return touchFirmware, touchMiscellaneous
}

// TouchRawReport copies the unfiltered coordinates from the most recently
// consumed sensing frame. Applications can use this for diagnostics without
// allowing controller artifacts into their input path.
func TouchRawReport(points []TouchPoint) int {
	count := touchLastRawCount
	if count > len(points) {
		count = len(points)
	}
	for index := 0; index < count; index++ {
		points[index] = touchLastRawPoints[index]
	}
	return count
}

// TouchLastReportStats returns protocol-level evidence for the last report.
func TouchLastReportStats() TouchReportStats {
	return TouchReportStats{
		SensingCounter: touchSensingCounter,
		Advanced:       touchLastAdvanced,
		RawCount:       touchLastRawCount,
		Checksum:       touchChecksum,
		CoordSum:       touchChecksumCoordSum,
		ReportSum:      touchChecksumReportSum,
		CoordXOR:       touchChecksumCoordXOR,
		Reports:        touchReports,
	}
}

// TouchInterruptLevels returns the GPIO23 levels sampled around the most
// recent report transaction. ST7121 asserts this data-ready signal low.
func TouchInterruptLevels() (int, int) {
	return touchIRQBefore, touchIRQAfter
}

// TouchReadFailure is one when the combined sensing/report transaction fails.
func TouchReadFailure() int {
	return touchReadFailure
}

// LandscapePoint rotates a native touch coordinate into 1280 by 720 space.
func LandscapePoint(point TouchPoint) (int, int) {
	return DisplayHeight - 1 - point.Y, point.X
}

// PortraitPoint preserves the ST7121's native 720 by 1280 coordinates.
func PortraitPoint(point TouchPoint) (int, int) {
	return point.X, point.Y
}
