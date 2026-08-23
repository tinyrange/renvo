//go:build renvo

// Package math exposes IEEE-754 bit and classification operations.
package math

const MaxFloat32 = 0x1p127 * (1 + (1 - 0x1p-23))
const SmallestNonzeroFloat32 = 0x1p-149
const MaxFloat64 = 0x1p1023 * (1 + (1 - 0x1p-52))
const SmallestNonzeroFloat64 = 0x1p-1074

func renvo_runtime_Float32bits(value float32) uint32     { return 0 }
func renvo_runtime_Float32frombits(value uint32) float32 { return 0 }
func renvo_runtime_Float64bits(value float64) uint64     { return 0 }
func renvo_runtime_Float64frombits(value uint64) float64 { return 0 }
func Float32bits(f float32) uint32                       { return renvo_runtime_Float32bits(f) }
func Float32frombits(b uint32) float32                   { return renvo_runtime_Float32frombits(b) }
func Float64bits(f float64) uint64                       { return renvo_runtime_Float64bits(f) }
func Float64frombits(b uint64) float64                   { return renvo_runtime_Float64frombits(b) }
func Inf(sign int) float64 {
	if sign < 0 {
		return Float64frombits(uint64(1)<<63 | uint64(0x7ff)<<52)
	}
	return Float64frombits(uint64(0x7ff) << 52)
}
func NaN() float64         { return Float64frombits(uint64(0x7ff)<<52 | uint64(1)<<51) }
func IsNaN(f float64) bool { return f != f }
func IsInf(f float64, sign int) bool {
	bits := Float64bits(f)
	magnitude := bits &^ (uint64(1) << 63)
	if magnitude != uint64(0x7ff)<<52 {
		return false
	}
	return sign == 0 || sign > 0 && bits>>63 == 0 || sign < 0 && bits>>63 != 0
}
func Signbit(f float64) bool { return Float64bits(f)>>63 != 0 }
func Abs(f float64) float64  { return Float64frombits(Float64bits(f) &^ (uint64(1) << 63)) }
func Copysign(f, sign float64) float64 {
	return Float64frombits(Float64bits(f)&^(uint64(1)<<63) | Float64bits(sign)&(uint64(1)<<63))
}
