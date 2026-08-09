package apk

const debugRSAModulusHex = "00b3786c3a0ca6c4a30ecf7aa9d7aecea969c2886f0d02f4c3a923f5f16507ae88d51fc56bd90b3be61d787f79d48091eeccced3962517c2b87780e0efcf310bf00459ee137e2219fc8354d32f725b37b59e26d7ac79266f1ba551cf017b14e49b5343242d2999a9d0646f1cac3393d3151ad8c52eaf23fcd27de97fe8999c90dcaa3a31bd756f2d44d8822170b8487baa803d5b7e16d6e76bd8684c50d35aeecdb713d678c9e01f08b432064315799a1701321c81c8759fa0a938c7432ce7b6891f9a79bfef9c8560f75b3e314b7da3cbf8eabaad4e8f76bcb4a1bf07fa751447647bce9e4485656528f6075cd445f3621a3184ee8b9fd61ad62f14c2ae5338e3"

const debugRSAPrivateExponentHex = "0ac5adf6a759bf306bac27c302af7b0cba2208b4645b35667a6ed9229dc621ee13f4c33c74f0f2adc8e90efb0e0be58bd2d3e5e950dd509267ec89692623888190914bf3f43c8718c6a7e9c122f90caaa4627fcc2f680f7fe0f811c26ca92ab63eaa275aba9b7ee62e3bbe6f2b9807871b19c06ecbe90f6771a7f945c9d5419b3e0a5c599f076c70f4cf1af96fb597436955e480498fc98c41a0abfb758a026f02888d1a93d973b0b6cc3f565b73cfcd100edcc27e4656683f5fd6c071f2ce71a9b7cf2a1d81444e49d98a08f2d33bec9a09f013e1a4019b78f01ba2e554223b4ae1b83cf5a7ac2d6773ea49b7d509f9699f087c38ac809465875581a746f231"

const rsaLimbs = 64

func rsaSignSHA256(digest []byte) []byte {
	if len(digest) != 32 {
		return nil
	}
	digestInfoPrefix := []byte{
		0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01,
		0x65, 0x03, 0x04, 0x02, 0x01, 0x05, 0x00, 0x04, 0x20,
	}
	encoded := make([]byte, 256)
	encoded[0] = 0
	encoded[1] = 1
	separator := len(encoded) - len(digestInfoPrefix) - len(digest) - 1
	for i := 2; i < separator; i++ {
		encoded[i] = 0xff
	}
	encoded[separator] = 0
	at := separator + 1
	for i := 0; i < len(digestInfoPrefix); i++ {
		encoded[at+i] = digestInfoPrefix[i]
	}
	at += len(digestInfoPrefix)
	for i := 0; i < len(digest); i++ {
		encoded[at+i] = digest[i]
	}
	modulusBytes := decodeHex(debugRSAModulusHex)
	if len(modulusBytes) == 257 && modulusBytes[0] == 0 {
		modulusBytes = modulusBytes[1:]
	}
	exponentBytes := decodeHex(debugRSAPrivateExponentHex)
	if len(modulusBytes) != 256 || len(exponentBytes) != 256 {
		return nil
	}
	modulus := rsaBytesToLimbs(modulusBytes)
	message := rsaBytesToLimbs(encoded)
	n0Inverse := rsaMontgomeryInverse(modulus[0])
	r2 := make([]uint32, rsaLimbs)
	r2[0] = 1
	for i := 0; i < rsaLimbs*64; i++ {
		rsaDoubleMod(r2, modulus)
	}
	base := rsaMontgomeryMultiply(message, r2, modulus, n0Inverse)
	one := make([]uint32, rsaLimbs)
	one[0] = 1
	result := rsaMontgomeryMultiply(one, r2, modulus, n0Inverse)
	for i := 0; i < len(exponentBytes); i++ {
		for bit := 7; bit >= 0; bit-- {
			result = rsaMontgomeryMultiply(result, result, modulus, n0Inverse)
			if exponentBytes[i]&(1<<uint(bit)) != 0 {
				result = rsaMontgomeryMultiply(result, base, modulus, n0Inverse)
			}
		}
	}
	result = rsaMontgomeryMultiply(result, one, modulus, n0Inverse)
	return rsaLimbsToBytes(result)
}

func rsaMontgomeryInverse(value uint32) uint32 {
	inverse := uint32(1)
	for i := 0; i < 5; i++ {
		inverse *= 2 - value*inverse
	}
	return (inverse ^ 0xffffffff) + 1
}

func rsaMontgomeryMultiply(
	left []uint32, right []uint32, modulus []uint32, inverse uint32,
) []uint32 {
	temporary := make([]uint32, rsaLimbs*2+2)
	for i := 0; i < rsaLimbs; i++ {
		carry := uint64(0)
		for j := 0; j < rsaLimbs; j++ {
			at := i + j
			value := uint64(temporary[at]) +
				uint64(left[i])*uint64(right[j]) + carry
			temporary[at] = uint32(value)
			carry = value >> 32
		}
		rsaAddCarry(temporary, i+rsaLimbs, carry)
	}
	for i := 0; i < rsaLimbs; i++ {
		factor := uint32(uint64(temporary[i]) * uint64(inverse))
		carry := uint64(0)
		for j := 0; j < rsaLimbs; j++ {
			at := i + j
			value := uint64(temporary[at]) +
				uint64(factor)*uint64(modulus[j]) + carry
			temporary[at] = uint32(value)
			carry = value >> 32
		}
		rsaAddCarry(temporary, i+rsaLimbs, carry)
	}
	result := make([]uint32, rsaLimbs)
	for i := 0; i < rsaLimbs; i++ {
		result[i] = temporary[i+rsaLimbs]
	}
	if temporary[rsaLimbs*2] != 0 || rsaCompare(result, modulus) >= 0 {
		rsaSubtract(result, modulus)
	}
	return result
}

func rsaAddCarry(words []uint32, at int, carry uint64) {
	for carry != 0 {
		value := uint64(words[at]) + carry
		words[at] = uint32(value)
		carry = value >> 32
		at++
	}
}

func rsaDoubleMod(value []uint32, modulus []uint32) {
	carry := uint64(0)
	for i := 0; i < rsaLimbs; i++ {
		next := uint64(value[i])*2 + carry
		value[i] = uint32(next)
		carry = next >> 32
	}
	if carry != 0 || rsaCompare(value, modulus) >= 0 {
		rsaSubtract(value, modulus)
	}
}

func rsaCompare(left []uint32, right []uint32) int {
	for i := rsaLimbs - 1; i >= 0; i-- {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func rsaSubtract(value []uint32, subtrahend []uint32) {
	borrow := uint64(0)
	for i := 0; i < rsaLimbs; i++ {
		left := uint64(value[i])
		right := uint64(subtrahend[i]) + borrow
		value[i] = uint32(left - right)
		if left < right {
			borrow = 1
		} else {
			borrow = 0
		}
	}
}

func rsaBytesToLimbs(data []byte) []uint32 {
	result := make([]uint32, rsaLimbs)
	for i := 0; i < len(data); i++ {
		fromEnd := len(data) - 1 - i
		result[i/4] |= uint32(data[fromEnd]) << uint((i%4)*8)
	}
	return result
}

func rsaLimbsToBytes(words []uint32) []byte {
	result := make([]byte, rsaLimbs*4)
	for i := 0; i < len(result); i++ {
		word := words[i/4]
		result[len(result)-1-i] = byte(word >> uint((i%4)*8))
	}
	return result
}

func decodeHex(value string) []byte {
	if len(value)%2 != 0 {
		return nil
	}
	out := make([]byte, len(value)/2)
	for i := 0; i < len(out); i++ {
		high, highOK := hexNibble(value[i*2])
		low, lowOK := hexNibble(value[i*2+1])
		if !highOK || !lowOK {
			return nil
		}
		out[i] = high<<4 | low
	}
	return out
}

func hexNibble(value byte) (byte, bool) {
	if value >= '0' && value <= '9' {
		return value - '0', true
	}
	if value >= 'a' && value <= 'f' {
		return value - 'a' + 10, true
	}
	if value >= 'A' && value <= 'F' {
		return value - 'A' + 10, true
	}
	return 0, false
}
