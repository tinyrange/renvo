//go:build !renvo

package binary

type ByteOrder interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
	Uint64([]byte) uint64
	PutUint16([]byte, uint16)
	PutUint32([]byte, uint32)
	PutUint64([]byte, uint64)
}

type littleEndian struct{}
type bigEndian struct{}

var LittleEndian littleEndian
var BigEndian bigEndian

func (littleEndian) Uint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) Uint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func (littleEndian) Uint64(b []byte) uint64 {
	return uint64(LittleEndian.Uint32(b)) | uint64(LittleEndian.Uint32(b[4:]))<<32
}

func (littleEndian) PutUint16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func (littleEndian) PutUint32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func (littleEndian) PutUint64(b []byte, v uint64) {
	LittleEndian.PutUint32(b, uint32(v))
	LittleEndian.PutUint32(b[4:], uint32(v>>32))
}

func (bigEndian) Uint16(b []byte) uint16 {
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigEndian) Uint32(b []byte) uint32 {
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

func (bigEndian) Uint64(b []byte) uint64 {
	return uint64(BigEndian.Uint32(b[4:])) | uint64(BigEndian.Uint32(b))<<32
}

func (bigEndian) PutUint16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func (bigEndian) PutUint32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func (bigEndian) PutUint64(b []byte, v uint64) {
	BigEndian.PutUint32(b, uint32(v>>32))
	BigEndian.PutUint32(b[4:], uint32(v))
}

func PutUvarint(buf []byte, x uint64) int {
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return i + 1
}

func Uvarint(buf []byte) (uint64, int) {
	var x uint64
	var s uint
	for i := 0; i < len(buf); i++ {
		b := buf[i]
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return 0, -(i + 1)
			}
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}

func PutVarint(buf []byte, x int64) int {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return PutUvarint(buf, ux)
}

func Varint(buf []byte) (int64, int) {
	ux, n := Uvarint(buf)
	x := int64(ux >> 1)
	if ux&1 != 0 {
		x = ^x
	}
	return x, n
}

func AppendUvarint(buf []byte, x uint64) []byte {
	var tmp [10]byte
	n := PutUvarint(tmp[:], x)
	return append(buf, tmp[:n]...)
}

func AppendVarint(buf []byte, x int64) []byte {
	var tmp [10]byte
	n := PutVarint(tmp[:], x)
	return append(buf, tmp[:n]...)
}
