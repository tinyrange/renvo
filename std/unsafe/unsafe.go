//go:build renvo

package unsafe

type Pointer *byte

// Add returns ptr advanced by the byte offset len. The frontend accepts
// every integer offset type; the backend's pointer-width argument conversion
// preserves the low bits needed for address arithmetic.
func Add(ptr Pointer, len int) Pointer {
	return Pointer(uintptr(ptr) + uintptr(len))
}
