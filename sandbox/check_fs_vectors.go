package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"renvo.dev/device/usb/lowspeed"
)

func dump(name string, pid byte, payload []byte) {
	states := encodeFullSpeed(pid, payload)
	n := len(states)
	// The production decoder deliberately has a small fixed packet scratch
	// buffer.  It is still useful as an independent check for short packets;
	// longer descriptor packets are checked by the host USB stack.
	if len(payload) <= 8 {
		var check [64]byte
		_, decoded, valid := lowspeed.Decode(check[:], states)
		if !valid || decoded != len(payload) {
			fmt.Printf("%s self-check failed: decoded=%d valid=%v\n", name, decoded, valid)
		}
	}
	fmt.Printf("%s %d crc=%04x\n\t.irp symbol,", name, n-3, lowspeed.CRC16(payload))
	for i, state := range states[:n-3] {
		if i > 0 {
			fmt.Print(",")
		}
		if state == lowspeed.LineK {
			fmt.Print("t5")
		} else {
			fmt.Print("t4")
		}
	}
	fmt.Println()
}

func dumpHandshake(name string, pid byte) {
	states := make([]byte, 32)
	n := lowspeed.EncodeHandshake(states, pid)
	fmt.Printf("%s %d\n\t.irp symbol,", name, n-3)
	for i, state := range states[:n-3] {
		if i > 0 {
			fmt.Print(",")
		}
		if state == lowspeed.LineK {
			fmt.Print("t5")
		} else {
			fmt.Print("t4")
		}
	}
	fmt.Println()
}

func encodeFullSpeed(pid byte, payload []byte) []byte {
	packet := make([]byte, 1, len(payload)+3)
	packet[0] = pid
	packet = append(packet, payload...)
	crc := lowspeed.CRC16(payload)
	packet = append(packet, byte(crc), byte(crc>>8))

	states := make([]byte, 0, 8+len(packet)*8+len(packet)*2+3)
	states = append(states,
		lowspeed.LineK, lowspeed.LineJ, lowspeed.LineK, lowspeed.LineJ,
		lowspeed.LineK, lowspeed.LineJ, lowspeed.LineK, lowspeed.LineK,
	)
	line := byte(lowspeed.LineK)
	ones := uint8(0)
	for _, value := range packet {
		for bit := 0; bit < 8; bit++ {
			if value&1 == 0 {
				if line == lowspeed.LineJ {
					line = lowspeed.LineK
				} else {
					line = lowspeed.LineJ
				}
				ones = 0
			} else {
				ones++
			}
			states = append(states, line)
			if ones == 6 {
				if line == lowspeed.LineJ {
					line = lowspeed.LineK
				} else {
					line = lowspeed.LineJ
				}
				states = append(states, line)
				ones = 0
			}
			value >>= 1
		}
	}
	return append(states, lowspeed.LineSE0, lowspeed.LineSE0, lowspeed.LineJ)
}

func decodeAssembly(name string, source []byte) {
	decodeAssemblyBranch(name, source, 0)
}

func decodeAssemblyBranch(name string, source []byte, branch int) {
	start := regexp.MustCompile(`(?m)^` + name + `:`).FindIndex(source)
	if start == nil {
		panic("missing " + name)
	}
	block := source[start[0]:]
	if next := regexp.MustCompile(`(?m)^[a-zA-Z_][a-zA-Z0-9_]*:`).FindIndex(block[len(name)+1:]); next != nil {
		block = block[:len(name)+1+next[0]]
	}
	matches := regexp.MustCompile(`(?m)^\s*\.irp symbol,([^\n]+)`).FindAllSubmatch(block, -1)
	if branch >= len(matches) {
		panic("missing branch " + name)
	}
	match := matches[branch]
	if len(match) != 2 {
		panic("missing " + name)
	}
	fields := strings.Split(string(match[1]), ",")
	states := make([]byte, 0, len(fields)+3)
	for _, field := range fields {
		if field == "t5" {
			states = append(states, lowspeed.LineK)
		} else if field == "t4" {
			states = append(states, lowspeed.LineJ)
		} else {
			panic("bad symbol " + field)
		}
	}
	states = append(states, lowspeed.LineSE0, lowspeed.LineSE0, lowspeed.LineJ)
	var payload [64]byte
	pid, n, ok := lowspeed.Decode(payload[:], states)
	fmt.Printf("%s: states=%d pid=%02x payload=%x ok=%v\n", name, len(fields), pid, payload[:n], ok)
}

func main() {
	dump("first8", lowspeed.PIDData1, []byte{0x12, 0x01, 0x10, 0x01, 0, 0, 0, 64})
	dumpHandshake("stall", lowspeed.PIDStall)
	dump("full18", lowspeed.PIDData1, []byte{0x12, 0x01, 0x10, 0x01, 0, 0, 0, 64, 0xad, 0xde, 0xc6, 0, 0, 1, 0, 0, 0, 1})
	dump("qualifierFirst", lowspeed.PIDData1, []byte{0x0a, 0x06, 0x00, 0x02, 0, 0, 0, 8})
	dump("qualifierLast", lowspeed.PIDData0, []byte{0x01, 0x00})
	dump("qualifierFull", lowspeed.PIDData1, []byte{0x0a, 0x06, 0x00, 0x02, 0, 0, 0, 64, 1, 0})
	dump("middle", lowspeed.PIDData0, []byte{0xfe, 0xca, 0xc6, 0x00, 0x00, 0x01, 0x00, 0x00})
	dump("last", lowspeed.PIDData1, []byte{0x00, 0x01})
	dump("configFirst", lowspeed.PIDData1, []byte{0x09, 0x02, 0x09, 0x00, 0x00, 0x01, 0x00, 0x80})
	dump("configMiddle", lowspeed.PIDData0, []byte{0x32, 0x09, 0x04, 0x00, 0x00, 0x00, 0xff, 0x00})
	dump("configLast", lowspeed.PIDData1, []byte{0x00, 0x00})
	dump("configByte9", lowspeed.PIDData0, []byte{0x32})
	dump("configFull", lowspeed.PIDData1, []byte{0x09, 0x02, 0x09, 0x00, 0x00, 0x01, 0x00, 0x80, 0x32})
	source, err := os.ReadFile("sandbox/esp32c6-fs-nak-sie.S")
	if err != nil {
		panic(err)
	}
	decodeAssembly("transmit_data1_8", source)
	decodeAssembly("transmit_data1_18", source)
	decodeAssembly("transmit_data1_18_fixed", source)
	decodeAssembly("transmit_data1_18_split", source)
	decodeAssembly("transmit_data0_8", source)
	decodeAssemblyBranch("transmit_data1_2", source, 2)
	decodeAssembly("transmit_config_data1_8", source)
	decodeAssembly("transmit_config_data0_8", source)
	decodeAssembly("transmit_config_data1_2", source)
	decodeAssembly("transmit_config_data0_1", source)
	decodeAssembly("transmit_qualifier_data1_8", source)
	decodeAssembly("transmit_qualifier_data0_2", source)
	decodeAssembly("transmit_qualifier_full", source)
	decodeAssembly("transmit_config_full", source)
}
