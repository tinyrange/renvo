package runtime

// Division is total and independent of host traps. Operands are truncated to
// width (32 or 64); results are zero-extended width-bit patterns. Signed division
// truncates toward zero. Overflow MIN/-1 yields MIN and remainder zero.
// A zero divisor yields an all-ones quotient and the original-width dividend
// as remainder. Guests with other policies must explicitly select or guard.
const (
	UnsignedDivide    = 38
	SignedDivide      = 39
	UnsignedRemainder = 40
	SignedRemainder   = 41
)

func divisionKind(k int) bool { return k >= UnsignedDivide && k <= SignedRemainder }
func validDivision(o Op) bool { return !divisionKind(o.Kind) || o.Imm == 32 || o.Imm == 64 }
func divisionOperation(kind int, a, b, width uint64) uint64 {
	mask := ^uint64(0)
	if width == 32 {
		mask = 0xffffffff
		a &= mask
		b &= mask
	}
	remainder := kind == UnsignedRemainder || kind == SignedRemainder
	if b == 0 {
		if remainder {
			return a
		}
		return mask
	}
	if kind == SignedDivide || kind == SignedRemainder {
		x, y := int64(a), int64(b)
		if width == 32 {
			x, y = int64(int32(a)), int64(int32(b))
		}
		if x == -9223372036854775807-1 && y == -1 {
			if remainder {
				return 0
			}
			return uint64(x) & mask
		}
		if remainder {
			return uint64(x%y) & mask
		}
		return uint64(x/y) & mask
	}
	if remainder {
		return a % b
	}
	return a / b
}
func (b *Builder) Divide(kind int, a, rhs Value, width int) Value {
	if divisionKind(kind) && (width == 32 || width == 64) && b.Ops[a].Kind == Const && b.Ops[rhs].Kind == Const {
		return b.Constant(divisionOperation(kind, b.Ops[a].Imm, b.Ops[rhs].Imm, uint64(width)))
	}
	return b.emit(Op{Kind: kind, A: a, B: rhs, Imm: uint64(width)})
}
