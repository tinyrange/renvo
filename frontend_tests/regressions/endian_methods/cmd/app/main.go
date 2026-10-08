package main

import "encoding/binary"

type Word struct{ Value int }
type Orders struct{ LittleEndian, BigEndian Word }

func check(order binary.ByteOrder, first, last byte) {
	data := make([]byte, 8)
	order.PutUint64(data, 0x8123456789abcdef)
	if data[0] != first || data[7] != last || order.Uint64(data) != 0x8123456789abcdef {
		panic("64-bit endian")
	}
	order.PutUint32(data, 0x89abcdef)
	if order.Uint32(data) != 0x89abcdef {
		panic("32-bit endian")
	}
	order.PutUint16(data, 0x89ab)
	if order.Uint16(data) != 0x89ab {
		panic("16-bit endian")
	}
}
func main() {
	check(binary.LittleEndian, 0xef, 0x81)
	check(binary.BigEndian, 0x81, 0xef)
	data := make([]byte, 8)
	put, get := binary.LittleEndian.PutUint64, binary.LittleEndian.Uint64
	put(data, 0xfedcba9876543210)
	if get(data) != 0xfedcba9876543210 {
		panic("endian method value")
	}
	orders := Orders{}
	orders.LittleEndian.Value, orders.BigEndian.Value = 3, 7
	if orders.LittleEndian.Value != 3 || orders.BigEndian.Value != 7 {
		panic("ordinary endian-named fields")
	}
	print("PASS\n")
}
