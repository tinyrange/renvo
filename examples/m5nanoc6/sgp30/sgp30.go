// Package sgp30 drives a Sensirion SGP30 connected to the M5NanoC6 Grove port.
package sgp30

import "renvo.dev/examples/m5nanoc6/board"

const address = byte(0x58)

func command(code uint16) bool {
	bytes := [2]byte{byte(code >> 8), byte(code)}
	return board.I2CWrite(address, bytes[:])
}

// Initialize starts the sensor's on-chip air-quality algorithm. After this,
// Measure must be called at one-second intervals for the baseline compensation
// algorithm to operate correctly.
func Initialize() bool {
	board.ConfigureI2C()
	if !command(0x2003) {
		return false
	}
	board.DelayMilliseconds(10)
	return true
}

func crc(first uint32, second uint32) uint32 {
	result := uint32(0xff)
	data := [2]uint32{first, second}
	for index := 0; index < 2; index++ {
		result = result ^ data[index]
		for bit := 0; bit < 8; bit++ {
			if result&0x80 != 0 {
				result = (result<<1 ^ 0x31) & 0xff
			} else {
				result = result << 1 & 0xff
			}
		}
	}
	return result
}

// Measure stores the total volatile organic compounds reading in ppb. Both
// response words must pass the sensor's CRC check, even though the SGP30's
// derived eCO2 word is intentionally not exposed.
func Measure(tvoc *uint16) bool {
	if !command(0x2008) {
		return false
	}
	// The specified measurement duration is at most 12 ms. Leave a small
	// margin before the read header so timer and oscillator tolerances cannot
	// turn a boundary read into a transient NACK.
	board.DelayMilliseconds(15)

	response := [6]uint32{}
	if !board.I2CRead6(address, &response) {
		return false
	}
	co2CRCValid := crc(response[0], response[1]) == response[2]
	tvocCRCValid := crc(response[3], response[4]) == response[5]
	if !co2CRCValid || !tvocCRCValid {
		return false
	}
	*tvoc = uint16(response[3])<<8 | uint16(response[4])
	return true
}
