// Package button provides polling debounce for digital switches.
package button

// Debouncer accepts a change only after the input stays stable for DelayMS.
// The initial state is released. Poll at least once per debounce interval.
type Debouncer struct {
	DelayMS   uint32
	Pressed   bool
	candidate bool
	since     uint32
}

// Update returns true on a debounced press or release. now is a wrapping
// millisecond counter; use delays shorter than half its range.
func (b *Debouncer) Update(pressed bool, now uint32) bool {
	if pressed != b.candidate {
		b.candidate = pressed
		b.since = now
	}
	if b.Pressed == b.candidate || now-b.since < b.DelayMS {
		return false
	}
	b.Pressed = b.candidate
	return true
}
