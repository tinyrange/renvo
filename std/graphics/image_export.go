package graphics

func appendImageText(output []byte, text string) []byte {
	for i := 0; i < len(text); i++ {
		output = append(output, text[i])
	}
	return output
}

func appendImageDecimal(output []byte, value int) []byte {
	if value == 0 {
		return append(output, '0')
	}
	var digits [20]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	for at < len(digits) {
		output = append(output, digits[at])
		at++
	}
	return output
}

// EncodePPM exports an RGBA8 image as a binary PPM image. PPM has no alpha
// channel, so the stored premultiplied RGB channels are emitted as-is. Invalid,
// destroyed and A8-only images return nil.
func (s *Surface) EncodePPM() []byte {
	if s == nil || s.Width <= 0 || s.Height <= 0 || s.Format != PixelRGBA8 || len(s.Pixels) < s.Stride*s.Height {
		return nil
	}
	output := make([]byte, 0, 32+s.Width*s.Height*3)
	output = appendImageText(output, "P6\n")
	output = appendImageDecimal(output, s.Width)
	output = append(output, ' ')
	output = appendImageDecimal(output, s.Height)
	output = appendImageText(output, "\n255\n")
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			offset := y*s.Stride + x*4
			output = append(output, s.Pixels[offset])
			output = append(output, s.Pixels[offset+1])
			output = append(output, s.Pixels[offset+2])
		}
	}
	return output
}
