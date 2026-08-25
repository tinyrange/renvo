// Package bits implements bit counting operations used by compact Renvo
// programs. More of math/bits can be added here as consumers require it.
package bits

// OnesCount8 returns the number of one bits in x.
func OnesCount8(x uint8) int {
	x = x - ((x >> 1) & 0x55)
	x = (x & 0x33) + ((x >> 2) & 0x33)
	return int((x + (x >> 4)) & 0x0f)
}
