package sha256

const Size = 32
const BlockSize = 64

var initial = [8]uint32{
	0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
	0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
}

var round = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

func rotateRight(x uint32, n uint) uint32 { return x>>n | x<<(32-n) }

func block(state *[8]uint32, data []byte) {
	var words [64]uint32
	for i := 0; i < 16; i++ {
		j := i * 4
		words[i] = uint32(data[j])<<24 | uint32(data[j+1])<<16 | uint32(data[j+2])<<8 | uint32(data[j+3])
	}
	for i := 16; i < 64; i++ {
		x := words[i-15]
		y := words[i-2]
		s0 := rotateRight(x, 7) ^ rotateRight(x, 18) ^ (x >> 3)
		s1 := rotateRight(y, 17) ^ rotateRight(y, 19) ^ (y >> 10)
		words[i] = words[i-16] + s0 + words[i-7] + s1
	}
	a, b, c, d := state[0], state[1], state[2], state[3]
	e, f, g, h := state[4], state[5], state[6], state[7]
	for i := 0; i < 64; i++ {
		s1 := rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
		choice := (e & f) ^ ((e ^ uint32(0xffffffff)) & g)
		t1 := h + s1 + choice + round[i] + words[i]
		s0 := rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
		majority := (a & b) ^ (a & c) ^ (b & c)
		t2 := s0 + majority
		h, g, f, e, d, c, b, a = g, f, e, d+t1, c, b, a, t1+t2
	}
	state[0] += a
	state[1] += b
	state[2] += c
	state[3] += d
	state[4] += e
	state[5] += f
	state[6] += g
	state[7] += h
}

func Sum256(data []byte) [Size]byte {
	totalLength := uint64(len(data))
	state := initial
	for len(data) >= BlockSize {
		block(&state, data[:BlockSize])
		data = data[BlockSize:]
	}
	paddingLength := BlockSize
	if len(data) >= 56 {
		paddingLength += BlockSize
	}
	padding := make([]byte, paddingLength)
	copy(padding, data)
	padding[len(data)] = 0x80
	bitLength := totalLength * 8
	for i := 0; i < 8; i++ {
		padding[len(padding)-1-i] = byte(bitLength)
		bitLength >>= 8
	}
	for len(padding) > 0 {
		block(&state, padding[:BlockSize])
		padding = padding[BlockSize:]
	}
	var sum [Size]byte
	for i, word := range state {
		j := i * 4
		sum[j] = byte(word >> 24)
		sum[j+1] = byte(word >> 16)
		sum[j+2] = byte(word >> 8)
		sum[j+3] = byte(word)
	}
	return sum
}
