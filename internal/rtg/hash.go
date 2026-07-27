package rtg

// sha256Bytes is a small self-hostable SHA-256 implementation. Keeping it in
// the definition core avoids a dependency on crypto/sha256, which is not part
// of the Renvo standard-library surface used by the self-hosted generator.
func sha256Bytes(source []byte) [32]byte {
	state := [8]uint32{
		0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
		0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
	}
	length := len(source)
	total := length + 1 + 8
	if total%64 != 0 {
		total += 64 - total%64
	}
	block := make([]byte, total)
	copy(block, source)
	block[length] = 0x80
	bits := uint64(length) * 8
	for i := 0; i < 8; i++ {
		block[total-1-i] = byte(bits)
		bits >>= 8
	}
	for offset := 0; offset < len(block); offset += 64 {
		sha256Block(&state, block[offset:offset+64])
	}
	var result [32]byte
	for i := 0; i < len(state); i++ {
		result[i*4] = byte(state[i] >> 24)
		result[i*4+1] = byte(state[i] >> 16)
		result[i*4+2] = byte(state[i] >> 8)
		result[i*4+3] = byte(state[i])
	}
	return result
}

func sha256Block(state *[8]uint32, block []byte) {
	k := [64]uint32{
		0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
		0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
		0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
		0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
		0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
		0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
		0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
		0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
		0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
		0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
		0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
		0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
		0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
		0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
		0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
		0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
	}
	var words [64]uint32
	for i := 0; i < 16; i++ {
		at := i * 4
		word := uint32(block[at]) << 24
		word |= uint32(block[at+1]) << 16
		word |= uint32(block[at+2]) << 8
		word |= uint32(block[at+3])
		words[i] = word
	}
	for i := 16; i < 64; i++ {
		s0 := rotateRight(words[i-15], 7) ^ rotateRight(words[i-15], 18) ^ words[i-15]>>3
		s1 := rotateRight(words[i-2], 17) ^ rotateRight(words[i-2], 19) ^ words[i-2]>>10
		words[i] = words[i-16] + s0 + words[i-7] + s1
	}
	a := (*state)[0]
	b := (*state)[1]
	c := (*state)[2]
	d := (*state)[3]
	e := (*state)[4]
	f := (*state)[5]
	g := (*state)[6]
	h := (*state)[7]
	for i := 0; i < 64; i++ {
		s1 := rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
		choice := (e & f) ^ ((e ^ 0xffffffff) & g)
		temp1 := h + s1 + choice + k[i] + words[i]
		s0 := rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
		majority := (a & b) ^ (a & c) ^ (b & c)
		temp2 := s0 + majority
		h = g
		g = f
		f = e
		e = d + temp1
		d = c
		c = b
		b = a
		a = temp1 + temp2
	}
	(*state)[0] += a
	(*state)[1] += b
	(*state)[2] += c
	(*state)[3] += d
	(*state)[4] += e
	(*state)[5] += f
	(*state)[6] += g
	(*state)[7] += h
}

func rotateRight(value uint32, count uint) uint32 {
	return value>>count | value<<(32-count)
}

func HashText(hash [32]byte) string {
	const digits = "0123456789abcdef"
	text := make([]byte, 64)
	for i := 0; i < len(hash); i++ {
		text[i*2] = digits[hash[i]>>4]
		text[i*2+1] = digits[hash[i]&15]
	}
	return string(text)
}
