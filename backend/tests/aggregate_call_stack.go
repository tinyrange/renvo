package main

type callPacket struct {
	data  [80]byte
	first int
	last  int
}

var globalCallPacket callPacket

func checkCallPacket(a int, b int, c int, packet callPacket, d int, e int, f int, other callPacket) bool {
	if a != 11 || b != 22 || c != 33 || d != 44 || e != 55 || f != 66 {
		return false
	}
	if packet.first != 17 || packet.last != 91 || other.first != 23 || other.last != 89 {
		return false
	}
	for i := 0; i < 80; i++ {
		if packet.data[i] != byte(i*13+7) || other.data[i] != byte(i*19+3) {
			return false
		}
	}
	packet.data[0] = 0
	other.last = 0
	return true
}

func callPacketArgument(packet callPacket, marker int) callPacket {
	if marker != 55 {
		panic("nested argument")
	}
	return packet
}

func appMain() int {
	var packet callPacket
	packet.first = 17
	packet.last = 91
	globalCallPacket.first = 23
	globalCallPacket.last = 89
	for i := 0; i < 80; i++ {
		packet.data[i] = byte(i*13 + 7)
		globalCallPacket.data[i] = byte(i*19 + 3)
	}
	if !checkCallPacket(11, 22, 33, packet, 44, 55, 66, callPacketArgument(globalCallPacket, 55)) {
		panic("aggregate arguments")
	}
	if packet.data[0] != 7 || globalCallPacket.last != 89 {
		panic("argument alias")
	}
	print("PASS\n")
	return 0
}
