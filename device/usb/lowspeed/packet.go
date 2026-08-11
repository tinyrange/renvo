// Package lowspeed implements the wire encoding shared by software USB PHYs.
package lowspeed

const (
	// LineSE0 drives both USB lines low.
	LineSE0 byte = iota
	// LineJ is the low-speed idle state: D+ low and D- high.
	LineJ
	// LineK is the opposite low-speed differential state.
	LineK

	PIDOut   = byte(0xe1)
	PIDIn    = byte(0x69)
	PIDSetup = byte(0x2d)
	PIDData0 = byte(0xc3)
	PIDData1 = byte(0x4b)
	PIDAck   = byte(0xd2)
	PIDNak   = byte(0x5a)
	PIDStall = byte(0x1e)
)

// CRC16 returns the complemented, reflected USB data CRC.
func CRC16(data []byte) uint16 {
	crc := uint16(0xffff)
	for _, value := range data {
		for bit := 0; bit < 8; bit++ {
			mix := (crc ^ uint16(value)) & 1
			crc >>= 1
			if mix != 0 {
				crc ^= 0xa001
			}
			value >>= 1
		}
	}
	return crc ^ 0xffff
}

// CRC5 returns the complemented, reflected USB token CRC for eleven bits.
func CRC5(value uint16) byte {
	crc := byte(0x1f)
	for bit := 0; bit < 11; bit++ {
		mix := (crc ^ byte(value)) & 1
		crc >>= 1
		if mix != 0 {
			crc ^= 0x14
		}
		value >>= 1
	}
	return (crc ^ 0x1f) & 0x1f
}

func validPID(pid byte) bool {
	return pid>>4 == (pid^0xff)&0x0f
}

func encodeBytes(destination []byte, bytes []byte) int {
	if len(destination) < 11 {
		return 0
	}
	sync := [8]byte{LineK, LineJ, LineK, LineJ, LineK, LineJ, LineK, LineK}
	copy(destination, sync[:])
	count := len(sync)
	line := LineK
	ones := uint8(0)
	for _, value := range bytes {
		for bit := 0; bit < 8; bit++ {
			if value&1 == 0 {
				if line == LineJ {
					line = LineK
				} else {
					line = LineJ
				}
				ones = 0
			} else {
				ones++
			}
			if count >= len(destination) {
				return 0
			}
			destination[count] = line
			count++
			if ones == 6 {
				if line == LineJ {
					line = LineK
				} else {
					line = LineJ
				}
				if count >= len(destination) {
					return 0
				}
				destination[count] = line
				count++
				ones = 0
			}
			value >>= 1
		}
	}
	if count+3 > len(destination) {
		return 0
	}
	destination[count], destination[count+1], destination[count+2] = LineSE0, LineSE0, LineJ
	return count + 3
}

// EncodeData writes one complete low-speed DATA0 or DATA1 packet as line
// states. The destination receives one state per bit cell, including SYNC and
// EOP. It returns zero if pid is invalid or destination is too small.
func EncodeData(destination []byte, pid byte, payload []byte) int {
	if !validPID(pid) {
		return 0
	}
	low := pid & 0x0f
	if low != 0x03 && low != 0x0b {
		return 0
	}
	if len(payload) > 8 {
		return 0
	}
	var packet [11]byte
	packet[0] = pid
	copy(packet[1:], payload)
	crc := CRC16(payload)
	packet[len(payload)+1] = byte(crc)
	packet[len(payload)+2] = byte(crc >> 8)
	return encodeBytes(destination, packet[:len(payload)+3])
}

// EncodeToken writes an OUT, IN, or SETUP token packet.
func EncodeToken(destination []byte, pid, address, endpoint byte) int {
	if !validPID(pid) || pid != PIDOut && pid != PIDIn && pid != PIDSetup || address > 127 || endpoint > 15 {
		return 0
	}
	value := uint16(address) | uint16(endpoint)<<7
	packet := [3]byte{pid, byte(value), byte(value >> 8)}
	packet[2] |= CRC5(value) << 3
	return encodeBytes(destination, packet[:])
}

// EncodeHandshake writes ACK, NAK, or STALL as a complete packet.
func EncodeHandshake(destination []byte, pid byte) int {
	if pid != PIDAck && pid != PIDNak && pid != PIDStall {
		return 0
	}
	packet := [1]byte{pid}
	return encodeBytes(destination, packet[:])
}

// Decode validates one complete low-speed waveform. Data receives the packet
// bytes after the PID and excludes its CRC. Token data is returned as the two
// packed token bytes without CRC bits. Handshakes return no data.
func Decode(data []byte, states []byte) (pid byte, count int, ok bool) {
	if len(states) < 11 {
		return 0, 0, false
	}
	sync := [8]byte{LineK, LineJ, LineK, LineJ, LineK, LineJ, LineK, LineK}
	for index, want := range sync {
		if states[index] != want {
			return 0, 0, false
		}
	}
	var bytes [16]byte
	byteCount, bitCount := 0, 0
	line, ones := byte(LineK), byte(0)
	index := 8
	for index < len(states) && states[index] != LineSE0 {
		next := states[index]
		if next != LineJ && next != LineK {
			return 0, 0, false
		}
		bit := byte(0)
		if next == line {
			bit = 1
		}
		line = next
		index++
		if ones == 6 {
			if bit != 0 {
				return 0, 0, false
			}
			ones = 0
			continue
		}
		if byteCount >= len(bytes) {
			return 0, 0, false
		}
		bytes[byteCount] |= bit << uint(bitCount)
		bitCount++
		if bit == 0 {
			ones = 0
		} else {
			ones++
		}
		if bitCount == 8 {
			bitCount = 0
			byteCount++
		}
	}
	if bitCount != 0 || index+2 >= len(states) || states[index] != LineSE0 || states[index+1] != LineSE0 || states[index+2] != LineJ || index+3 != len(states) || byteCount == 0 {
		return 0, 0, false
	}
	pid = bytes[0]
	if !validPID(pid) {
		return 0, 0, false
	}
	low := pid & 0x0f
	if low == 0x03 || low == 0x0b {
		if byteCount < 3 || byteCount-3 > len(data) {
			return 0, 0, false
		}
		count = byteCount - 3
		crc := uint16(bytes[byteCount-2]) | uint16(bytes[byteCount-1])<<8
		wantCRC := uint16(0xffff)
		for crcIndex := 1; crcIndex < byteCount-2; crcIndex++ {
			value := bytes[crcIndex]
			for crcBit := 0; crcBit < 8; crcBit++ {
				mix := (wantCRC ^ uint16(value)) & 1
				wantCRC >>= 1
				if mix != 0 {
					wantCRC ^= 0xa001
				}
				value >>= 1
			}
		}
		wantCRC ^= 0xffff
		if crc != wantCRC {
			return 0, 0, false
		}
		copy(data, bytes[1:1+count])
		return pid, count, true
	}
	if low == 0x01 || low == 0x09 || low == 0x0d {
		if byteCount != 3 || len(data) < 2 {
			return 0, 0, false
		}
		value := uint16(bytes[1]) | uint16(bytes[2]&7)<<8
		if bytes[2]>>3 != CRC5(value) {
			return 0, 0, false
		}
		data[0], data[1] = bytes[1], bytes[2]&7
		return pid, 2, true
	}
	if pid == PIDAck || pid == PIDNak || pid == PIDStall {
		return pid, 0, byteCount == 1
	}
	return 0, 0, false
}
