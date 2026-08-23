package main

import (
	"math"
	"strconv"
)

func main() {
	decimal, err := strconv.ParseFloat("0.02", 64)
	if err != nil || math.Float64bits(decimal) != 0x3f947ae147ae147b {
		print("FAIL: decimal parse\n")
		return
	}
	hexadecimal, err := strconv.ParseFloat("0x1.8p+2", 64)
	if err != nil || hexadecimal != 6 {
		print("FAIL: hexadecimal parse\n")
		return
	}
	binary32, err := strconv.ParseFloat("0.1", 32)
	if err != nil || math.Float32bits(float32(binary32)) != 0x3dcccccd {
		print("FAIL: binary32 parse\n")
		return
	}
	values := [5]float64{0.02, 0.2, -12.5, 1e-7, 1.2345678901234567}
	for i := 0; i < len(values); i++ {
		text := strconv.FormatFloat(values[i], 'g', -1, 64)
		roundTrip, parseErr := strconv.ParseFloat(text, 64)
		if parseErr != nil || math.Float64bits(roundTrip) != math.Float64bits(values[i]) {
			print("FAIL: round trip\n")
			return
		}
	}
	if strconv.FormatFloat(0.02, 'g', -1, 64) != "0.02" || strconv.FormatFloat(-12.5, 'g', -1, 64) != "-12.5" {
		print("FAIL: shortest formatting\n")
		return
	}
	smallest, smallestErr := strconv.ParseFloat("5e-324", 64)
	if smallestErr != nil || math.Float64bits(smallest) != 1 {
		print("FAIL: smallest parse\n")
		return
	}
	largest, largestErr := strconv.ParseFloat("1.7976931348623157e308", 64)
	if largestErr != nil || math.Float64bits(largest) != 0x7fefffffffffffff {
		print("FAIL: largest parse\n")
		return
	}
	if strconv.FormatFloat(math.Float64frombits(1), 'g', -1, 64) != "5e-324" || strconv.FormatFloat(math.Float64frombits(0x7fefffffffffffff), 'g', -1, 64) != "1.7976931348623157e+308" {
		print("FAIL: float64 boundary formatting\n")
		return
	}
	if strconv.FormatFloat(float64(math.Float32frombits(1)), 'g', -1, 32) != "1e-45" || strconv.FormatFloat(float64(math.Float32frombits(0x7f7fffff)), 'g', -1, 32) != "3.4028235e+38" {
		print("FAIL: float32 boundary formatting\n")
		return
	}
	negativeZero := math.Float64frombits(uint64(1) << 63)
	if strconv.FormatFloat(negativeZero, 'g', -1, 64) != "-0" {
		print("FAIL: signed zero formatting\n")
		return
	}
	parsedZero, zeroErr := strconv.ParseFloat("-0", 64)
	if zeroErr != nil || math.Float64bits(parsedZero) != uint64(1)<<63 {
		print("FAIL: signed zero parsing\n")
		return
	}
	halfway, halfwayErr := strconv.ParseFloat("0x1.00000000000018p0", 64)
	if halfwayErr != nil || math.Float64bits(halfway) != 0x3ff0000000000002 {
		print("FAIL: hexadecimal rounding\n")
		return
	}
	if strconv.FormatFloat(12.345, 'f', 2, 64) != "12.35" || strconv.FormatFloat(12.345, 'e', 2, 64) != "1.23e+01" || strconv.FormatFloat(12.345, 'g', 3, 64) != "12.3" {
		print("FAIL: precision formatting\n")
		return
	}
	if _, syntaxErr := strconv.ParseFloat("1.2.3", 64); syntaxErr == nil {
		print("FAIL: syntax error\n")
		return
	}
	print("PASS\n")
}
