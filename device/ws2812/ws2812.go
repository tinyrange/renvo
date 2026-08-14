// Package ws2812 provides a portable driver for WS2812-compatible RGB pixels
// and strips, including three-channel SK6812 devices.
package ws2812

// RGB is one red, green and blue addressable-LED value.
type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

// Transmitter sends a complete WS2812 wire-order byte stream. Implementations
// are responsible for the 800 kHz waveform and reset interval.
type Transmitter interface {
	Transmit([]byte) bool
}

// Strip is a WS2812-compatible pixel chain backed by a hardware transmitter.
type Strip struct {
	transmitter Transmitter
	bytes       []byte
}

// New returns a strip backed by transmitter.
func New(transmitter Transmitter) Strip {
	return Strip{transmitter: transmitter}
}

func encode(pixels []RGB, reuse []byte) []byte {
	required := len(pixels) * 3
	if cap(reuse) < required {
		reuse = make([]byte, required)
	} else {
		reuse = reuse[:required]
	}
	for i := 0; i < len(pixels); i++ {
		pixel := pixels[i]
		reuse[i*3] = pixel.Green
		reuse[i*3+1] = pixel.Red
		reuse[i*3+2] = pixel.Blue
	}
	return reuse
}

// SetPixels emits pixels in WS2812/SK6812 GRB wire order. It returns false if
// the hardware transmitter reports an error or misses its deadline.
func (s *Strip) SetPixels(pixels []RGB) bool {
	s.bytes = encode(pixels, s.bytes)
	return s.transmitter.Transmit(s.bytes)
}

// Set emits one red, green and blue pixel.
func (s *Strip) Set(red, green, blue uint8) {
	pixel := [1]RGB{{Red: red, Green: green, Blue: blue}}
	s.SetPixels(pixel[:])
}
