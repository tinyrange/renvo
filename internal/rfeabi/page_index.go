package rfeabi

// PageIndex selects a bounded cache slot, never access authority. Every lookup
// must still compare the full raw page number and live permissions. Hashing
// here avoids permuting architectural addresses in every generated memory op.
func (c *Context) PageIndex(number uint64) uint64 {
	if c.HashPages != 0 {
		number ^= number >> 6
	}
	return number & 63
}
