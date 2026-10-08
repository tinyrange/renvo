package runtime

// Rich integer records survive optimization until host instruction selection.
const (
	UnsignedMulHigh = 27
	SignedMulHigh   = 28
	LeadingZeros    = 29
	LeadingSigns    = 30
	ReverseBytes    = 31 // Imm: width in low byte, byte-reversal group width above it
	ReverseBits     = 32
)

func validRich(o Op) bool {
	if o.Kind >= LeadingZeros && o.Kind <= ReverseBits {
		width, group := o.Imm&255, o.Imm>>8
		if width != 32 && width != 64 {
			return false
		}
		if o.Kind == ReverseBytes {
			return group == 16 || group == 32 || group == 64 && width == 64
		}
		return group == 0
	}
	return true
}
func richOperation(kind int, a, b, encoding uint64) uint64 {
	if kind == UnsignedMulHigh || kind == SignedMulHigh {
		al, ah := a&0xffffffff, a>>32
		bl, bh := b&0xffffffff, b>>32
		low, cross1, cross2 := al*bl, ah*bl, al*bh
		middle := (low >> 32) + (cross1 & 0xffffffff) + (cross2 & 0xffffffff)
		hi := ah*bh + (cross1 >> 32) + (cross2 >> 32) + (middle >> 32)
		if kind == SignedMulHigh {
			if int64(a) < 0 {
				hi -= b
			}
			if int64(b) < 0 {
				hi -= a
			}
		}
		return hi
	}
	width := encoding & 255
	if width == 32 {
		a = uint64(uint32(a))
	}
	switch kind {
	case LeadingZeros, LeadingSigns:
		if kind == LeadingSigns && a>>(width-1) != 0 {
			a = ^a
		}
		count := uint64(0)
		for bit := width; bit > 0; bit-- {
			if a&(uint64(1)<<(bit-1)) != 0 {
				break
			}
			count++
		}
		if kind == LeadingSigns {
			count--
		}
		return count
	case ReverseBits:
		result := uint64(0)
		for bit := uint64(0); bit < width; bit++ {
			result = (result << 1) | ((a >> bit) & 1)
		}
		return result
	case ReverseBytes:
		group := encoding >> 8
		var result uint64
		for offset := uint64(0); offset < width; offset += group {
			for j := uint64(0); j < group; j += 8 {
				result |= ((a >> (offset + j)) & 255) << (offset + group - 8 - j)
			}
		}
		return result
	}
	return 0
}
func (b *Builder) MultiplyHigh(a, rhs Value, signed bool) Value {
	kind := UnsignedMulHigh
	if signed {
		kind = SignedMulHigh
	}
	if b.Ops[a].Kind == Const && b.Ops[rhs].Kind == Const {
		return b.Constant(richOperation(kind, b.Ops[a].Imm, b.Ops[rhs].Imm, 0))
	}
	return b.emit(Op{Kind: kind, A: a, B: rhs})
}
func (b *Builder) RichUnary(kind int, a Value, width, group int) Value {
	encoding := uint64(width) | uint64(group)<<8
	if b.Ops[a].Kind == Const && validRich(Op{Kind: kind, Imm: encoding}) {
		return b.Constant(richOperation(kind, b.Ops[a].Imm, 0, encoding))
	}
	return b.emit(Op{Kind: kind, A: a, B: a, Imm: encoding})
}
