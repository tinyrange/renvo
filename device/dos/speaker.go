package dos

func SpeakerOn(frequency int) {
	if frequency <= 0 {
		SpeakerOff()
		return
	}
	SpeakerDivisor(speakerDivisor(frequency))
}

// SpeakerDivisor programs PIT channel 2 directly.
func SpeakerDivisor(divisor uint16) {
	if divisor == 0 {
		divisor = 1
	}
	portOut8(0x43, 0xb6)
	portOut8(0x42, byte(divisor))
	portOut8(0x42, byte(divisor>>8))
	portOut8(0x61, portIn8(0x61)|3)
}

// speakerDivisor divides the 21-bit PIT clock (0x12:34de) without requiring a
// native integer wider than the target's 16-bit int.
func speakerDivisor(frequency int) uint16 {
	if frequency < 19 {
		return 0xffff
	}
	if frequency > 32767 {
		frequency = 32767
	}
	denominator := uint16(frequency)
	var remainder uint16
	var quotient uint16
	for bit := 20; bit >= 0; bit-- {
		input := uint16(0)
		if bit >= 16 {
			input = uint16(0x12>>uint(bit-16)) & 1
		} else {
			input = uint16(0x34de>>uint(bit)) & 1
		}
		remainder = remainder<<1 | input
		if remainder >= denominator {
			remainder -= denominator
			if bit < 16 {
				quotient |= uint16(1) << uint(bit)
			}
		}
	}
	return quotient
}

func SpeakerOff() { portOut8(0x61, portIn8(0x61)&^3) }
