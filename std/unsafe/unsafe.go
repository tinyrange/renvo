//go:build renvo

package unsafe

type Pointer *byte

// Add returns ptr advanced by the byte offset len. The frontend accepts
// every integer offset type; the backend's pointer-width argument conversion
// preserves the low bits needed for address arithmetic.
func Add(ptr Pointer, len int) Pointer {
	return Pointer(uintptr(ptr) + uintptr(len))
}

type stringHeader struct {
	data   *byte
	length int
}

// String exposes length bytes at ptr without copying. The uint64 parameter
// retains wide integer arguments until they have been checked against the
// target's int range; negative signed arguments convert to out-of-range values.
func String(ptr *byte, length uint64) string {
	if length > uint64(^uint(0)>>1) || ptr == nil && length != 0 {
		panic("unsafe.String: invalid length or nil pointer")
	}
	header := stringHeader{data: ptr, length: int(length)}
	return *(*string)(Pointer(&header))
}

// StringData returns the address of the string's original bytes.
func StringData(value string) *byte {
	return (*stringHeader)(Pointer(&value)).data
}
