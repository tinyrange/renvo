//go:build !renvo

// Package math exposes the IEEE-754 helpers used by Renvo programs.
package math

import stdmath "math"

const MaxFloat32 = stdmath.MaxFloat32
const SmallestNonzeroFloat32 = stdmath.SmallestNonzeroFloat32
const MaxFloat64 = stdmath.MaxFloat64
const SmallestNonzeroFloat64 = stdmath.SmallestNonzeroFloat64

func Float32bits(f float32) uint32     { return stdmath.Float32bits(f) }
func Float32frombits(b uint32) float32 { return stdmath.Float32frombits(b) }
func Float64bits(f float64) uint64     { return stdmath.Float64bits(f) }
func Float64frombits(b uint64) float64 { return stdmath.Float64frombits(b) }
func Inf(sign int) float64             { return stdmath.Inf(sign) }
func NaN() float64                     { return stdmath.NaN() }
func IsNaN(f float64) bool             { return stdmath.IsNaN(f) }
func IsInf(f float64, sign int) bool   { return stdmath.IsInf(f, sign) }
func Signbit(f float64) bool           { return stdmath.Signbit(f) }
func Abs(f float64) float64            { return stdmath.Abs(f) }
func Copysign(f, sign float64) float64 { return stdmath.Copysign(f, sign) }
