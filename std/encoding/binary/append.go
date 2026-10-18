package binary

func (littleEndian) AppendUint16(b []byte, v uint16) []byte { return append(b, byte(v), byte(v>>8)) }
func (littleEndian) AppendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
func (littleEndian) AppendUint64(b []byte, v uint64) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}
func (bigEndian) AppendUint16(b []byte, v uint16) []byte { return append(b, byte(v>>8), byte(v)) }
func (bigEndian) AppendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
func (bigEndian) AppendUint64(b []byte, v uint64) []byte {
	return append(b, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
