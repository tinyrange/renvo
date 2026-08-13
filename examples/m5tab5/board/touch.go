package board

const touchAddress = byte(0x55)

// TouchPoint is one ST7121 contact in the native 720 by 1280 orientation.
type TouchPoint struct {
	ID       int
	X        int
	Y        int
	Strength int
}

var touchMaximum = 10
var touchReadFailure int

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
	var info [10]byte
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
	if info[0] != 1 {
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
	var advanced [1]byte
	if !i2cReadRegister(touchSDA, touchSCL, touchAddress, 0x0010, advanced[:]) {
		touchReadFailure = 1
		return 0, false
	}
	if advanced[0]&0x08 == 0 {
		return 0, true
	}
	var reports [70]byte
	length := touchMaximum * 7
	if !i2cReadRegister(touchSDA, touchSCL, touchAddress, 0x0014, reports[:length]) {
		touchReadFailure = 2
		return 0, false
	}
	count := 0
	for contact := 0; contact < touchMaximum && count < len(points); contact++ {
		offset := contact * 7
		if reports[offset]&0x80 == 0 {
			continue
		}
		points[count].ID = contact
		points[count].X = int(reports[offset]&0x3f)<<8 | int(reports[offset+1])
		points[count].Y = int(reports[offset+2])<<8 | int(reports[offset+3])
		points[count].Strength = int(reports[offset+4])
		// Ignore transient records outside the geometry reported by firmware.
		// Passing them to Forms creates clipped markers along the display edge.
		if points[count].X < 0 || points[count].X >= DisplayWidth ||
			points[count].Y < 0 || points[count].Y >= DisplayHeight {
			continue
		}
		count++
	}
	return count, true
}

// TouchReadFailure identifies the failed polling phase for serial diagnostics:
// one is the advanced-status byte and two is the coordinate report.
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
