//go:build renvo

package strconv

import "math"

// ParseFloat converts a decimal or hexadecimal floating-point string to an
// IEEE-754 value. It accepts the same special values and exponent forms as Go's
// strconv.ParseFloat and rounds to binary32 when bitSize is 32.
func ParseFloat(s string, bitSize int) (float64, error) {
	if bitSize != 32 && bitSize != 64 {
		bitSize = 64
	}
	if s == "NaN" || s == "+NaN" || s == "-NaN" {
		return math.NaN(), nil
	}
	negative := false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		negative = s[0] == '-'
		s = s[1:]
	}
	if s == "Inf" || s == "Infinity" {
		return math.Inf(boolSign(negative)), nil
	}
	if len(s) == 0 {
		return 0, ErrSyntax
	}
	var value float64
	var ok bool
	nonzero := false
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		_, nonzero, ok = parseHexFloat(s[2:])
	} else {
		_, nonzero, ok = parseDecimalFloat(s)
	}
	if !ok {
		return 0, ErrSyntax
	}
	bits := uint64(0)
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		bits = parseExactFloatBits(s, bitSize)
	} else {
		bits = parseDecimalFloatBits(s, bitSize)
	}
	if bitSize == 32 {
		bits32 := uint32(bits)
		if negative {
			bits32 |= uint32(1) << 31
		}
		value = float64(math.Float32frombits(bits32))
	} else {
		if negative {
			bits |= uint64(1) << 63
		}
		value = math.Float64frombits(bits)
	}
	if math.IsInf(value, 0) || nonzero && value == 0 {
		return value, ErrRange
	}
	return value, nil
}

// floatDecimal is a decimal digit buffer used for correctly rounded binary
// conversion. Its operations only multiply and divide by powers of two, which
// keeps the implementation portable to targets without native 64-bit divide.
type floatDecimal struct {
	digit [800]byte
	nd    int
	dp    int
	trunc bool
}

func floatDecimalTrim(value *floatDecimal) {
	for value.nd > 0 && value.digit[value.nd-1] == '0' {
		value.nd--
	}
	if value.nd == 0 {
		value.dp = 0
	}
}

func floatDecimalAssign(value *floatDecimal, number uint64) {
	var reverse [24]byte
	count := 0
	for number > 0 {
		quotient := number / 10
		reverse[count] = byte(number-10*quotient) + '0'
		count++
		number = quotient
	}
	value.nd = 0
	for count > 0 {
		count--
		value.digit[value.nd] = reverse[count]
		value.nd++
	}
	value.dp = value.nd
	value.trunc = false
	floatDecimalTrim(value)
}

func floatDecimalSet(value *floatDecimal, s string) {
	value.nd = 0
	value.dp = 0
	value.trunc = false
	afterDot := false
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == 'e' || ch == 'E' {
			break
		}
		if ch == '.' {
			afterDot = true
			value.dp = value.nd
		} else if ch >= '0' && ch <= '9' {
			if ch == '0' && value.nd == 0 {
				if afterDot {
					value.dp--
				}
			} else if value.nd < len(value.digit) {
				value.digit[value.nd] = ch
				value.nd++
			} else if ch != '0' {
				value.trunc = true
			}
		}
		i++
	}
	if !afterDot {
		value.dp = value.nd
	}
	if i < len(s) {
		i++
		sign := 1
		if i < len(s) && s[i] == '-' {
			sign = -1
			i++
		} else if i < len(s) && s[i] == '+' {
			i++
		}
		exponent := 0
		for i < len(s) {
			ch := s[i]
			if ch >= '0' && ch <= '9' && exponent < 10000 {
				exponent = exponent*10 + int(ch-'0')
			}
			i++
		}
		value.dp += sign * exponent
	}
	floatDecimalTrim(value)
}

func floatDecimalRightShift(value *floatDecimal, shift uint32) {
	read := 0
	write := 0
	number := uint32(0)
	for number>>shift == 0 {
		if read >= value.nd {
			if number == 0 {
				value.nd = 0
				return
			}
			for number>>shift == 0 {
				number *= 10
				read++
			}
			break
		}
		number = number*10 + uint32(value.digit[read]-'0')
		read++
	}
	value.dp -= read - 1
	mask := uint32(1)<<shift - 1
	for read < value.nd {
		digit := number >> shift
		number &= mask
		value.digit[write] = byte(digit + '0')
		write++
		number = number*10 + uint32(value.digit[read]-'0')
		read++
	}
	for number > 0 {
		digit := number >> shift
		number &= mask
		if write < len(value.digit) {
			value.digit[write] = byte(digit + '0')
			write++
		} else if digit != 0 {
			value.trunc = true
		}
		number *= 10
	}
	value.nd = write
	floatDecimalTrim(value)
}

type floatDecimalLeftShift struct {
	delta  int
	cutoff string
}

var floatDecimalLeftShifts = [29]floatDecimalLeftShift{
	{0, ""},
	{1, "5"},
	{1, "25"},
	{1, "125"},
	{2, "625"},
	{2, "3125"},
	{2, "15625"},
	{3, "78125"},
	{3, "390625"},
	{3, "1953125"},
	{4, "9765625"},
	{4, "48828125"},
	{4, "244140625"},
	{4, "1220703125"},
	{5, "6103515625"},
	{5, "30517578125"},
	{5, "152587890625"},
	{6, "762939453125"},
	{6, "3814697265625"},
	{6, "19073486328125"},
	{7, "95367431640625"},
	{7, "476837158203125"},
	{7, "2384185791015625"},
	{7, "11920928955078125"},
	{8, "59604644775390625"},
	{8, "298023223876953125"},
	{8, "1490116119384765625"},
	{9, "7450580596923828125"},
	{9, "37252902984619140625"},
}

func floatDecimalPrefixLess(digits []byte, cutoff string) bool {
	for i := 0; i < len(cutoff); i++ {
		if i >= len(digits) {
			return true
		}
		if digits[i] != cutoff[i] {
			return digits[i] < cutoff[i]
		}
	}
	return false
}

func floatDecimalLeftShiftBits(value *floatDecimal, shift uint32) {
	delta := floatDecimalLeftShifts[shift].delta
	if floatDecimalPrefixLess(value.digit[:value.nd], floatDecimalLeftShifts[shift].cutoff) {
		delta--
	}
	read := value.nd
	write := value.nd + delta
	number := uint32(0)
	for read > 0 {
		read--
		number += uint32(value.digit[read]-'0') << shift
		quotient := number / 10
		remainder := number - 10*quotient
		write--
		if write < len(value.digit) {
			value.digit[write] = byte(remainder + '0')
		} else if remainder != 0 {
			value.trunc = true
		}
		number = quotient
	}
	for number > 0 {
		quotient := number / 10
		remainder := number - 10*quotient
		write--
		if write < len(value.digit) {
			value.digit[write] = byte(remainder + '0')
		} else if remainder != 0 {
			value.trunc = true
		}
		number = quotient
	}
	value.nd += delta
	if value.nd > len(value.digit) {
		value.nd = len(value.digit)
	}
	value.dp += delta
	floatDecimalTrim(value)
}

func floatDecimalShift(value *floatDecimal, shift int) {
	if value.nd == 0 {
		return
	}
	for shift > 28 {
		floatDecimalLeftShiftBits(value, 28)
		shift -= 28
	}
	for shift > 0 {
		floatDecimalLeftShiftBits(value, uint32(shift))
		shift = 0
	}
	for shift < -28 {
		floatDecimalRightShift(value, 28)
		shift += 28
	}
	if shift < 0 {
		floatDecimalRightShift(value, uint32(-shift))
	}
}

func floatDecimalShouldRoundUp(value *floatDecimal, digits int) bool {
	if digits < 0 || digits >= value.nd {
		return false
	}
	if value.digit[digits] == '5' && digits+1 == value.nd {
		if value.trunc {
			return true
		}
		return digits > 0 && (value.digit[digits-1]-'0')&1 != 0
	}
	return value.digit[digits] >= '5'
}

func floatDecimalRoundDown(value *floatDecimal, digits int) {
	if digits < 0 || digits >= value.nd {
		return
	}
	value.nd = digits
	floatDecimalTrim(value)
}

func floatDecimalRoundUp(value *floatDecimal, digits int) {
	if digits < 0 || digits >= value.nd {
		return
	}
	for i := digits - 1; i >= 0; i-- {
		if value.digit[i] < '9' {
			value.digit[i]++
			value.nd = i + 1
			return
		}
	}
	value.digit[0] = '1'
	value.nd = 1
	value.dp++
}

func floatDecimalRound(value *floatDecimal, digits int) {
	if digits < 0 || digits >= value.nd {
		return
	}
	if floatDecimalShouldRoundUp(value, digits) {
		floatDecimalRoundUp(value, digits)
	} else {
		floatDecimalRoundDown(value, digits)
	}
}

func floatDecimalRoundedInteger(value *floatDecimal) uint64 {
	if value.dp > 20 {
		return ^uint64(0)
	}
	index := 0
	number := uint64(0)
	for index < value.dp && index < value.nd {
		number = number*10 + uint64(value.digit[index]-'0')
		index++
	}
	for index < value.dp {
		number *= 10
		index++
	}
	if floatDecimalShouldRoundUp(value, value.dp) {
		number++
	}
	return number
}

func parseDecimalFloatBits(s string, bitSize int) uint64 {
	var value floatDecimal
	floatDecimalSet(&value, s)
	mantissaBits, exponentBits, bias := 52, 11, -1023
	if bitSize == 32 {
		mantissaBits, exponentBits, bias = 23, 8, -127
	}
	if value.nd == 0 {
		return 0
	}
	if value.dp > 310 {
		return uint64((1<<exponentBits)-1) << mantissaBits
	}
	if value.dp < -330 {
		return 0
	}
	powers := [9]int{1, 3, 6, 9, 13, 16, 19, 23, 26}
	exponent := 0
	for value.dp > 0 {
		shift := 27
		if value.dp < len(powers) {
			shift = powers[value.dp]
		}
		floatDecimalShift(&value, -shift)
		exponent += shift
	}
	for value.dp < 0 || value.dp == 0 && value.digit[0] < '5' {
		shift := 27
		if -value.dp < len(powers) {
			shift = powers[-value.dp]
		}
		floatDecimalShift(&value, shift)
		exponent -= shift
	}
	exponent--
	if exponent < bias+1 {
		shift := bias + 1 - exponent
		floatDecimalShift(&value, -shift)
		exponent += shift
	}
	if exponent-bias >= 1<<exponentBits-1 {
		return uint64((1<<exponentBits)-1) << mantissaBits
	}
	floatDecimalShift(&value, 1+mantissaBits)
	mantissa := floatDecimalRoundedInteger(&value)
	if mantissa == uint64(2)<<mantissaBits {
		mantissa >>= 1
		exponent++
		if exponent-bias >= 1<<exponentBits-1 {
			return uint64((1<<exponentBits)-1) << mantissaBits
		}
	}
	if mantissa&(uint64(1)<<mantissaBits) == 0 {
		exponent = bias
	}
	bits := mantissa & (uint64(1)<<mantissaBits - 1)
	bits |= uint64((exponent-bias)&(1<<exponentBits-1)) << mantissaBits
	return bits
}

type exactFloatBig struct {
	word   [80]uint32
	length int
}

func exactFloatBigSet(value *exactFloatBig, number uint32) {
	value.length = 0
	if number != 0 {
		value.word[0] = number
		value.length = 1
	}
}

func exactFloatBigCopy(dst *exactFloatBig, src *exactFloatBig) {
	dst.length = src.length
	for i := 0; i < src.length; i++ {
		dst.word[i] = src.word[i]
	}
}

func exactFloatBigMulSmall(value *exactFloatBig, multiplier uint32) bool {
	carry := uint32(0)
	for i := 0; i < value.length; i++ {
		word := value.word[i]
		low := (word&0xffff)*multiplier + carry
		high := (word>>16)*multiplier + (low >> 16)
		value.word[i] = low&0xffff | high<<16
		carry = high >> 16
	}
	if carry != 0 {
		if value.length >= len(value.word) {
			return false
		}
		value.word[value.length] = carry
		value.length++
	}
	return true
}

func exactFloatUint32Less(left uint32, right uint32) bool {
	return int32(left^uint32(0x80000000)) < int32(right^uint32(0x80000000))
}

func exactFloatBigAddSmall(value *exactFloatBig, addend uint32) bool {
	if value.length == 0 {
		exactFloatBigSet(value, addend)
		return true
	}
	carry := addend
	for i := 0; carry != 0 && i < value.length; i++ {
		word := value.word[i]
		value.word[i] = word + carry
		carry = 0
		if exactFloatUint32Less(value.word[i], word) {
			carry = 1
		}
	}
	if carry != 0 {
		if value.length >= len(value.word) {
			return false
		}
		value.word[value.length] = carry
		value.length++
	}
	return true
}

func exactFloatBigBitLength(value *exactFloatBig) int {
	if value.length == 0 {
		return 0
	}
	word := value.word[value.length-1]
	bits := 0
	for word != 0 {
		word >>= 1
		bits++
	}
	return (value.length-1)*32 + bits
}

func exactFloatBigShiftedWord(value *exactFloatBig, index int, shift int) uint32 {
	wordShift := shift / 32
	bitShift := shift & 31
	source := index - wordShift
	result := uint32(0)
	if source >= 0 && source < value.length {
		result = value.word[source] << bitShift
	}
	if bitShift != 0 && source-1 >= 0 && source-1 < value.length {
		result |= value.word[source-1] >> (32 - bitShift)
	}
	return result
}

func exactFloatBigShiftedLength(value *exactFloatBig, shift int) int {
	if value.length == 0 {
		return 0
	}
	length := value.length + shift/32
	if shift&31 != 0 && value.word[value.length-1]>>(32-(shift&31)) != 0 {
		length++
	}
	return length
}

func exactFloatBigCompareShifted(left *exactFloatBig, right *exactFloatBig, rightShift int) int {
	leftLength := left.length
	rightLength := exactFloatBigShiftedLength(right, rightShift)
	if leftLength < rightLength {
		return -1
	}
	if leftLength > rightLength {
		return 1
	}
	for i := leftLength - 1; i >= 0; i-- {
		rightWord := exactFloatBigShiftedWord(right, i, rightShift)
		if exactFloatUint32Less(left.word[i], rightWord) {
			return -1
		}
		if exactFloatUint32Less(rightWord, left.word[i]) {
			return 1
		}
	}
	return 0
}

func exactFloatBigSubShifted(left *exactFloatBig, right *exactFloatBig, rightShift int) {
	borrow := uint32(0)
	for i := 0; i < left.length; i++ {
		word := left.word[i]
		subtrahend := exactFloatBigShiftedWord(right, i, rightShift)
		next := word - subtrahend
		borrowFromWord := exactFloatUint32Less(word, subtrahend)
		withBorrow := next - borrow
		borrowFromCarry := exactFloatUint32Less(next, borrow)
		left.word[i] = withBorrow
		borrow = 0
		if borrowFromWord || borrowFromCarry {
			borrow = 1
		}
	}
	for left.length > 0 && left.word[left.length-1] == 0 {
		left.length--
	}
}

func exactFloatBigShiftLeft(value *exactFloatBig, shift int) bool {
	if value.length == 0 || shift == 0 {
		return true
	}
	wordShift := shift / 32
	newLength := exactFloatBigShiftedLength(value, shift)
	if newLength > len(value.word) {
		return false
	}
	for i := newLength - 1; i >= 0; i-- {
		value.word[i] = exactFloatBigShiftedWord(value, i, shift)
	}
	for i := 0; i < wordShift; i++ {
		value.word[i] = 0
	}
	value.length = newLength
	return true
}

func exactFloatBigQuotientRounded(numerator *exactFloatBig, denominator *exactFloatBig, sticky bool) (uint32, uint32) {
	var remainder exactFloatBig
	exactFloatBigCopy(&remainder, numerator)
	difference := exactFloatBigBitLength(&remainder) - exactFloatBigBitLength(denominator)
	quotientLow := uint32(0)
	quotientHigh := uint32(0)
	for bit := difference; bit >= 0; bit-- {
		quotientHigh = quotientHigh<<1 | quotientLow>>31
		quotientLow <<= 1
		if exactFloatBigCompareShifted(&remainder, denominator, bit) >= 0 {
			exactFloatBigSubShifted(&remainder, denominator, bit)
			quotientLow |= 1
		}
	}
	comparison := exactFloatBigCompareShifted(denominator, &remainder, 1)
	if comparison < 0 || comparison == 0 && (sticky || quotientLow&1 != 0) {
		quotientLow++
		if quotientLow == 0 {
			quotientHigh++
		}
	}
	return quotientLow, quotientHigh
}

func exactFloatBigRatioExponent(numerator *exactFloatBig, denominator *exactFloatBig) int {
	difference := exactFloatBigBitLength(numerator) - exactFloatBigBitLength(denominator)
	if difference >= 0 {
		if exactFloatBigCompareShifted(numerator, denominator, difference) < 0 {
			difference--
		}
		return difference
	}
	if exactFloatBigCompareShifted(denominator, numerator, -difference) > 0 {
		difference--
	}
	return difference
}

func exactFloatBitsFromWords(high uint32, low uint32) uint64 {
	return uint64(high)<<32 | uint64(low)
}

func exactFloatInfinityBits(fractionBits int, exponentBits int) uint64 {
	if fractionBits == 23 {
		return uint64(uint32((1<<exponentBits)-1) << 23)
	}
	return exactFloatBitsFromWords(uint32((1<<exponentBits)-1)<<20, 0)
}

func exactFloatBigEncode(numerator *exactFloatBig, denominator *exactFloatBig, binaryShift int, fractionBits int, exponentBits int, bias int, sticky bool) uint64 {
	if numerator.length == 0 {
		return 0
	}
	maxExponent := (1 << exponentBits) - 2 - bias
	minExponent := 1 - bias
	exponent := exactFloatBigRatioExponent(numerator, denominator) + binaryShift
	if exponent > maxExponent {
		return exactFloatInfinityBits(fractionBits, exponentBits)
	}
	var scaledNumerator exactFloatBig
	var scaledDenominator exactFloatBig
	exactFloatBigCopy(&scaledNumerator, numerator)
	exactFloatBigCopy(&scaledDenominator, denominator)
	targetExponent := exponent - fractionBits
	if exponent < minExponent {
		targetExponent = minExponent - fractionBits
	}
	shift := binaryShift - targetExponent
	if shift >= 0 {
		if !exactFloatBigShiftLeft(&scaledNumerator, shift) {
			return exactFloatInfinityBits(fractionBits, exponentBits)
		}
	} else if !exactFloatBigShiftLeft(&scaledDenominator, -shift) {
		return 0
	}
	significandLow, significandHigh := exactFloatBigQuotientRounded(&scaledNumerator, &scaledDenominator, sticky)
	if exponent < minExponent {
		if fractionBits == 23 {
			if significandHigh != 0 || !exactFloatUint32Less(significandLow, uint32(1)<<23) {
				return uint64(uint32(1) << 23)
			}
			return uint64(significandLow)
		}
		if significandHigh >= uint32(1)<<20 {
			return exactFloatBitsFromWords(uint32(1)<<20, 0)
		}
		return exactFloatBitsFromWords(significandHigh, significandLow)
	}
	overflow := significandHigh != 0
	if fractionBits == 52 {
		overflow = significandHigh >= uint32(1)<<21
	} else if fractionBits == 23 {
		overflow = significandHigh != 0 || !exactFloatUint32Less(significandLow, uint32(1)<<24)
	}
	if overflow {
		significandLow = significandLow>>1 | significandHigh<<31
		significandHigh >>= 1
		exponent++
		if exponent > maxExponent {
			return exactFloatInfinityBits(fractionBits, exponentBits)
		}
	}
	if fractionBits == 23 {
		return uint64(uint32(exponent+bias)<<23 | significandLow&(uint32(1)<<23-1))
	}
	high := uint32(exponent+bias)<<20 | significandHigh&(uint32(1)<<20-1)
	return exactFloatBitsFromWords(high, significandLow)
}

func parseExactFloatBits(s string, bitSize int) uint64 {
	fractionBits, exponentBits, bias := 52, 11, 1023
	if bitSize == 32 {
		fractionBits, exponentBits, bias = 23, 8, 127
	}
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return parseExactHexFloatBits(s[2:], fractionBits, exponentBits, bias)
	}
	return 0
}

func parseExactHexFloatBits(s string, fractionBits int, exponentBits int, bias int) uint64 {
	var numerator exactFloatBig
	var denominator exactFloatBig
	exactFloatBigSet(&numerator, 0)
	exactFloatBigSet(&denominator, 1)
	fractionDigits := 0
	afterDot := false
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == 'p' || ch == 'P' {
			break
		}
		if ch == '.' {
			afterDot = true
		} else if ch != '_' {
			digit, valid := digitValue(ch)
			if valid {
				exactFloatBigMulSmall(&numerator, 16)
				exactFloatBigAddSmall(&numerator, uint32(digit))
				if afterDot {
					fractionDigits++
				}
			}
		}
		i++
	}
	exponent2 := -fractionDigits * 4
	if i < len(s) {
		i++
		sign := 1
		if i < len(s) && s[i] == '-' {
			sign = -1
			i++
		} else if i < len(s) && s[i] == '+' {
			i++
		}
		exponent := 0
		for i < len(s) {
			ch := s[i]
			if ch >= '0' && ch <= '9' && exponent < 10000 {
				exponent = exponent*10 + int(ch-'0')
			}
			i++
		}
		exponent2 += sign * exponent
	}
	return exactFloatBigEncode(&numerator, &denominator, exponent2, fractionBits, exponentBits, bias, false)
}

func parseDecimalFloat(s string) (float64, bool, bool) {
	value := float64(0)
	digits := 0
	fractionDigits := 0
	droppedDigits := 0
	firstDropped := -1
	seenDot := false
	seenDigit := false
	nonzero := false
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '_' {
			if i == 0 || i+1 >= len(s) || s[i-1] < '0' || s[i-1] > '9' || s[i+1] < '0' || s[i+1] > '9' {
				return 0, false, false
			}
			i++
			continue
		}
		if c == '.' && !seenDot {
			seenDot = true
			i++
			continue
		}
		if c == 'e' || c == 'E' {
			break
		}
		if c < '0' || c > '9' {
			return 0, false, false
		}
		seenDigit = true
		digit := int(c - '0')
		if digit != 0 {
			nonzero = true
		}
		if seenDot {
			fractionDigits++
		}
		if digits < 18 {
			value = value*10 + float64(digit)
			digits++
		} else {
			droppedDigits++
			if firstDropped < 0 {
				firstDropped = digit
			}
		}
		i++
	}
	if !seenDigit {
		return 0, false, false
	}
	exponent := 0
	if i < len(s) {
		i++
		sign := 1
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			if s[i] == '-' {
				sign = -1
			}
			i++
		}
		start := i
		for i < len(s) {
			c := s[i]
			if c == '_' {
				if i == start || i+1 >= len(s) || s[i-1] < '0' || s[i-1] > '9' || s[i+1] < '0' || s[i+1] > '9' {
					return 0, false, false
				}
				i++
				continue
			}
			if c < '0' || c > '9' {
				return 0, false, false
			}
			if exponent < 10000 {
				exponent = exponent*10 + int(c-'0')
			}
			i++
		}
		if i == start {
			return 0, false, false
		}
		exponent *= sign
	}
	if firstDropped > 5 || firstDropped == 5 {
		value++
	}
	exponent += droppedDigits - fractionDigits
	return scaleFloat10(value, exponent), nonzero, true
}

func parseHexFloat(s string) (float64, bool, bool) {
	value := float64(0)
	fractionDigits := 0
	seenDot := false
	seenDigit := false
	nonzero := false
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '_' {
			i++
			continue
		}
		if c == '.' && !seenDot {
			seenDot = true
			i++
			continue
		}
		if c == 'p' || c == 'P' {
			break
		}
		digit, valid := digitValue(c)
		if !valid || digit >= 16 {
			return 0, false, false
		}
		seenDigit = true
		if digit != 0 {
			nonzero = true
		}
		value = value*16 + float64(digit)
		if seenDot {
			fractionDigits++
		}
		i++
	}
	if !seenDigit || i >= len(s) {
		return 0, false, false
	}
	i++
	exponent := 0
	sign := 1
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}
	start := i
	for i < len(s) {
		c := s[i]
		if c == '_' {
			i++
			continue
		}
		if c < '0' || c > '9' {
			return 0, false, false
		}
		if exponent < 10000 {
			exponent = exponent*10 + int(c-'0')
		}
		i++
	}
	if i == start {
		return 0, false, false
	}
	exponent = sign*exponent - fractionDigits*4
	return scaleFloat2(value, exponent), nonzero, true
}

func scaleFloat10(value float64, exponent int) float64 {
	powers := [9]float64{1e1, 1e2, 1e4, 1e8, 1e16, 1e32, 1e64, 1e128, 1e256}
	negative := exponent < 0
	if negative {
		exponent = -exponent
	}
	for bit := 0; exponent != 0; bit++ {
		if exponent&1 != 0 {
			if bit >= len(powers) {
				if negative {
					return 0
				}
				return math.Inf(1)
			}
			if negative {
				value /= powers[bit]
			} else {
				value *= powers[bit]
			}
		}
		exponent >>= 1
	}
	return value
}

func scaleFloat2(value float64, exponent int) float64 {
	if exponent > 1200 {
		return math.Inf(1)
	}
	if exponent < -1200 {
		return 0
	}
	for exponent >= 64 {
		value *= 0x1p64
		exponent -= 64
	}
	for exponent <= -64 {
		value *= 0x1p-64
		exponent += 64
	}
	if exponent > 0 {
		for exponent > 0 {
			value *= 2
			exponent--
		}
	} else {
		for exponent < 0 {
			value *= 0.5
			exponent++
		}
	}
	return value
}

// FormatFloat converts an IEEE-754 value to decimal or hexadecimal text. The
// -1 precision for g/G selects the shortest candidate that round-trips through
// ParseFloat at the requested bit size.
func FormatFloat(f float64, format byte, prec int, bitSize int) string {
	if bitSize == 32 {
		f = float64(float32(f))
	} else {
		bitSize = 64
	}
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "+Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}
	negative := math.Signbit(f)
	if negative {
		f = -f
	}
	if format == 'x' || format == 'X' {
		return formatHexFloat(f, negative, format, prec, bitSize)
	}
	if format != 'e' && format != 'E' && format != 'f' && format != 'g' && format != 'G' {
		return "%" + oneByteString(format)
	}
	return formatExactDecimalFloat(f, negative, format, prec, bitSize)
}

func floatDecimalRoundShortest(value *floatDecimal, mantissa uint64, exponent int, mantissaBits int, bias int) {
	if mantissa == 0 {
		value.nd = 0
		return
	}
	minimumExponent := bias + 1
	if exponent > minimumExponent && 332*(value.dp-value.nd) >= 100*(exponent-mantissaBits) {
		return
	}
	var upper floatDecimal
	floatDecimalAssign(&upper, mantissa*2+1)
	floatDecimalShift(&upper, exponent-mantissaBits-1)
	mantissaLow := uint64(0)
	exponentLow := 0
	if mantissa > uint64(1)<<mantissaBits || exponent == minimumExponent {
		mantissaLow = mantissa - 1
		exponentLow = exponent
	} else {
		mantissaLow = mantissa*2 - 1
		exponentLow = exponent - 1
	}
	var lower floatDecimal
	floatDecimalAssign(&lower, mantissaLow*2+1)
	floatDecimalShift(&lower, exponentLow-mantissaBits-1)
	inclusive := mantissa&1 == 0
	upperDelta := byte(0)
	for upperIndex := 0; ; upperIndex++ {
		middleIndex := upperIndex - upper.dp + value.dp
		if middleIndex >= value.nd {
			break
		}
		lowerIndex := upperIndex - upper.dp + lower.dp
		lowerDigit := byte('0')
		if lowerIndex >= 0 && lowerIndex < lower.nd {
			lowerDigit = lower.digit[lowerIndex]
		}
		middleDigit := byte('0')
		if middleIndex >= 0 {
			middleDigit = value.digit[middleIndex]
		}
		upperDigit := byte('0')
		if upperIndex < upper.nd {
			upperDigit = upper.digit[upperIndex]
		}
		canRoundDown := lowerDigit != middleDigit || inclusive && lowerIndex+1 == lower.nd
		if upperDelta == 0 && middleDigit+1 < upperDigit {
			upperDelta = 2
		} else if upperDelta == 0 && middleDigit != upperDigit {
			upperDelta = 1
		} else if upperDelta == 1 && (middleDigit != '9' || upperDigit != '0') {
			upperDelta = 2
		}
		canRoundUp := upperDelta > 0 && (inclusive || upperDelta > 1 || upperIndex+1 < upper.nd)
		if canRoundDown && canRoundUp {
			floatDecimalRound(value, middleIndex+1)
			return
		}
		if canRoundDown {
			floatDecimalRoundDown(value, middleIndex+1)
			return
		}
		if canRoundUp {
			floatDecimalRoundUp(value, middleIndex+1)
			return
		}
	}
}

func formatExactDecimalFloat(f float64, negative bool, format byte, precision int, bitSize int) string {
	bits := math.Float64bits(f)
	mantissaBits, exponentBits, bias := 52, 11, -1023
	if bitSize == 32 {
		bits = uint64(math.Float32bits(float32(f)))
		mantissaBits, exponentBits, bias = 23, 8, -127
	}
	exponent := int(bits>>mantissaBits) & (1<<exponentBits - 1)
	mantissa := bits & (uint64(1)<<mantissaBits - 1)
	if exponent == 0 {
		exponent++
	} else {
		mantissa |= uint64(1) << mantissaBits
	}
	exponent += bias
	var decimal floatDecimal
	floatDecimalAssign(&decimal, mantissa)
	floatDecimalShift(&decimal, exponent-mantissaBits)
	shortest := precision < 0
	if shortest {
		floatDecimalRoundShortest(&decimal, mantissa, exponent, mantissaBits, bias)
		switch format {
		case 'e', 'E':
			precision = decimal.nd - 1
			if precision < 0 {
				precision = 0
			}
		case 'f':
			precision = decimal.nd - decimal.dp
			if precision < 0 {
				precision = 0
			}
		case 'g', 'G':
			precision = decimal.nd
		}
	} else {
		switch format {
		case 'e', 'E':
			floatDecimalRound(&decimal, precision+1)
		case 'f':
			floatDecimalRound(&decimal, decimal.dp+precision)
		case 'g', 'G':
			if precision == 0 {
				precision = 1
			}
			floatDecimalRound(&decimal, precision)
		}
	}
	return renderFloatDecimal(&decimal, negative, format, precision, shortest)
}

func renderFloatDecimal(decimal *floatDecimal, negative bool, format byte, precision int, shortest bool) string {
	if format == 'g' || format == 'G' {
		exponentPrecision := precision
		if exponentPrecision > decimal.nd && decimal.nd >= decimal.dp {
			exponentPrecision = decimal.nd
		}
		if shortest {
			exponentPrecision = 6
		}
		exponent := decimal.dp - 1
		if exponent < -4 || exponent >= exponentPrecision {
			if precision > decimal.nd {
				precision = decimal.nd
			}
			marker := byte('e')
			if format == 'G' {
				marker = 'E'
			}
			return renderFloatDecimalExponent(decimal, negative, marker, precision-1)
		}
		if precision > decimal.dp {
			precision = decimal.nd
		}
		fractionPrecision := precision - decimal.dp
		if fractionPrecision < 0 {
			fractionPrecision = 0
		}
		return renderFloatDecimalFixed(decimal, negative, fractionPrecision)
	}
	if format == 'e' || format == 'E' {
		return renderFloatDecimalExponent(decimal, negative, format, precision)
	}
	return renderFloatDecimalFixed(decimal, negative, precision)
}

func renderFloatDecimalExponent(decimal *floatDecimal, negative bool, marker byte, precision int) string {
	out := make([]byte, 0, precision+8)
	if negative {
		out = append(out, '-')
	}
	first := byte('0')
	if decimal.nd != 0 {
		first = decimal.digit[0]
	}
	out = append(out, first)
	if precision > 0 {
		out = append(out, '.')
		for i := 1; i <= precision; i++ {
			digit := byte('0')
			if i < decimal.nd {
				digit = decimal.digit[i]
			}
			out = append(out, digit)
		}
	}
	out = append(out, marker)
	exponent := decimal.dp - 1
	if decimal.nd == 0 {
		exponent = 0
	}
	if exponent < 0 {
		out = append(out, '-')
		exponent = -exponent
	} else {
		out = append(out, '+')
	}
	if exponent < 10 {
		out = append(out, '0', byte(exponent)+'0')
	} else if exponent < 100 {
		out = append(out, byte(exponent/10)+'0', byte(exponent%10)+'0')
	} else {
		out = append(out, byte(exponent/100)+'0', byte(exponent/10%10)+'0', byte(exponent%10)+'0')
	}
	return string(out)
}

func renderFloatDecimalFixed(decimal *floatDecimal, negative bool, precision int) string {
	out := make([]byte, 0, decimal.nd+precision+4)
	if negative {
		out = append(out, '-')
	}
	if decimal.dp > 0 {
		index := 0
		for index < decimal.dp {
			digit := byte('0')
			if index < decimal.nd {
				digit = decimal.digit[index]
			}
			out = append(out, digit)
			index++
		}
	} else {
		out = append(out, '0')
	}
	if precision > 0 {
		out = append(out, '.')
		for i := 0; i < precision; i++ {
			digit := byte('0')
			index := decimal.dp + i
			if index >= 0 && index < decimal.nd {
				digit = decimal.digit[index]
			}
			out = append(out, digit)
		}
	}
	return string(out)
}

func shortenDecimal(digits []byte, exponent int, negative bool, bitSize int, value float64) []byte {
	best := digits
	want32 := uint32(0)
	want64 := uint64(0)
	if bitSize == 32 {
		want32 = math.Float32bits(float32(value))
	} else {
		want64 = math.Float64bits(value)
	}
	for length := len(digits) - 1; length >= 1; length-- {
		candidate := make([]byte, length)
		copy(candidate, digits[:length])
		matched := decimalCandidateMatches(candidate, exponent, negative, bitSize, want32, want64)
		if !matched {
			rounded := make([]byte, length)
			copy(rounded, candidate)
			rounded, _ = roundDecimalDigits(rounded, exponent)
			matched = decimalCandidateMatches(rounded, exponent, negative, bitSize, want32, want64)
			if matched {
				candidate = rounded
			}
		}
		if !matched {
			break
		}
		best = candidate
	}
	return best
}

func decimalCandidateMatches(digits []byte, exponent int, negative bool, bitSize int, want32 uint32, want64 uint64) bool {
	text := formatDecimalDigits(digits, exponent, 'g', -1)
	if negative {
		text = "-" + text
	}
	parsed, err := ParseFloat(text, bitSize)
	if err != nil {
		return false
	}
	if bitSize == 32 {
		return math.Float32bits(float32(parsed)) == want32
	}
	return math.Float64bits(parsed) == want64
}

func roundDecimalDigits(digits []byte, exponent int) ([]byte, int) {
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < '9' {
			digits[i]++
			return digits, exponent
		}
		digits[i] = '0'
	}
	out := make([]byte, len(digits))
	out[0] = '1'
	return out, exponent + 1
}

func formatDecimalDigits(digits []byte, exponent int, format byte, prec int) string {
	if format == 'e' || format == 'E' {
		out := string(digits[:1])
		if len(digits) > 1 {
			out += "." + string(digits[1:])
		} else if prec > 0 {
			out += "." + repeatByte('0', prec)
		}
		return out + oneByteString(format) + formatExponent(exponent)
	}
	if format == 'g' || format == 'G' {
		if exponent < -4 || exponent >= len(digits) {
			marker := byte('e')
			if format == 'G' {
				marker = 'E'
			}
			return formatDecimalDigits(digits, exponent, marker, -1)
		}
	}
	point := exponent + 1
	out := ""
	if point <= 0 {
		out = "0." + repeatByte('0', -point) + string(digits)
	} else if point >= len(digits) {
		out = string(digits) + repeatByte('0', point-len(digits))
	} else {
		out = string(digits[:point]) + "." + string(digits[point:])
	}
	if format == 'g' || format == 'G' {
		for len(out) > 0 && out[len(out)-1] == '0' && containsByte(out, '.') {
			out = out[:len(out)-1]
		}
		if len(out) > 0 && out[len(out)-1] == '.' {
			out = out[:len(out)-1]
		}
	}
	return out
}

func formatExponent(exponent int) string {
	sign := "+"
	if exponent < 0 {
		sign = "-"
		exponent = -exponent
	}
	digits := Itoa(exponent)
	if len(digits) < 2 {
		digits = "0" + digits
	}
	return sign + digits
}

func formatHexFloat(f float64, negative bool, format byte, prec int, bitSize int) string {
	bits := math.Float64bits(f)
	fractionBits := 52
	exponentBits := 11
	bias := 1023
	if bitSize == 32 {
		bits = uint64(math.Float32bits(float32(f)))
		fractionBits = 23
		exponentBits = 8
		bias = 127
	}
	exponentMask := uint64(1<<exponentBits) - 1
	exponent := int(bits>>fractionBits&exponentMask) - bias
	fraction := bits & (uint64(1)<<fractionBits - 1)
	if exponent == -bias {
		exponent++
	} else {
		fraction |= uint64(1) << fractionBits
	}
	digits := (fractionBits + 3) / 4
	if prec >= 0 && prec < digits {
		digits = prec
	}
	var out []byte
	if negative {
		out = append(out, '-')
	}
	out = append(out, '0', 'x', '1')
	if format == 'X' {
		out[len(out)-2] = 'X'
	}
	if digits > 0 {
		out = append(out, '.')
		for i := 0; i < digits; i++ {
			shift := fractionBits - 4*(i+1)
			nibble := uint64(0)
			if shift >= 0 {
				nibble = fraction >> shift & 15
			} else {
				nibble = fraction << (-shift) & 15
			}
			digit := byte('0') + byte(nibble)
			if nibble >= 10 {
				digit = byte('a') + byte(nibble-10)
				if format == 'X' {
					digit = byte('A') + byte(nibble-10)
				}
			}
			out = append(out, digit)
		}
	}
	marker := byte('p')
	if format == 'X' {
		marker = 'P'
	}
	out = append(out, marker)
	exponentText := formatExponent(exponent)
	for i := 0; i < len(exponentText); i++ {
		out = append(out, exponentText[i])
	}
	return string(out)
}

func repeatByte(value byte, count int) string {
	var out []byte
	for i := 0; i < count; i++ {
		out = append(out, value)
	}
	return string(out)
}

func oneByteString(value byte) string {
	var out []byte
	out = append(out, value)
	return string(out)
}

func containsByte(value string, want byte) bool {
	for i := 0; i < len(value); i++ {
		if value[i] == want {
			return true
		}
	}
	return false
}

func boolSign(negative bool) int {
	if negative {
		return -1
	}
	return 1
}

func AppendFloat(dst []byte, f float64, format byte, prec int, bitSize int) []byte {
	text := FormatFloat(f, format, prec, bitSize)
	for i := 0; i < len(text); i++ {
		dst = append(dst, text[i])
	}
	return dst
}
