package main

type Signed int32
type Alias = Signed
type Unsigned uint32

func main() {
	signed := make(map[Alias]int)
	unsigned := make(map[Unsigned]int)
	// Equal low bits, sign boundaries and overflowing multiplication exercise
	// every key bit through growth, collision chains and moved-entry deletion.
	for i := 0; i < 4096; i++ {
		key := uint32(i)*0x100000 + 0xfffff
		signed[Signed(int32(key))] = i
		unsigned[Unsigned(key)] = i
	}
	for i := 0; i < 4096; i += 3 {
		key := uint32(i)*0x100000 + 0xfffff
		delete(signed, Signed(int32(key)))
		delete(unsigned, Unsigned(key))
	}
	for i := 0; i < 4096; i++ {
		key := uint32(i)*0x100000 + 0xfffff
		s, sok := signed[Signed(int32(key))]
		u, uok := unsigned[Unsigned(key)]
		if i%3 == 0 {
			if sok || uok || s != 0 || u != 0 {
				panic("deleted")
			}
		} else if !sok || !uok || s != i || u != i {
			panic("lookup")
		}
	}
	for i := 0; i < 4096; i += 3 {
		key := uint32(i)*0x100000 + 0xfffff
		signed[Signed(int32(key))] = i
		unsigned[Unsigned(key)] = i
	}
	count := 0
	for key, value := range signed {
		if unsigned[Unsigned(uint32(key))] != value {
			panic("range")
		}
		count++
	}
	if count != 4096 || len(unsigned) != count {
		panic("length")
	}
	clear(signed)
	clear(unsigned)
	signed[-2147483648] = 0
	unsigned[0xffffffff] = 0
	if value, ok := signed[-2147483648]; !ok || value != 0 {
		panic("present zero")
	}
	if _, ok := unsigned[0xffffffff]; !ok {
		panic("unsigned boundary")
	}
	if _, ok := signed[2147483647]; ok {
		panic("absent boundary")
	}
	delete(signed, -2147483648)
	delete(unsigned, 0xffffffff)
	if len(signed) != 0 || len(unsigned) != 0 {
		panic("reuse")
	}

	smallSigned := make(map[int8]int)
	smallUnsigned := make(map[byte]int)
	mediumSigned := make(map[int16]int)
	mediumUnsigned := make(map[uint16]int)
	for i := 0; i < 256; i++ {
		smallSigned[int8(i)] = i
		smallUnsigned[byte(i)] = i
		mediumSigned[int16(i*256+255)] = i
		mediumUnsigned[uint16(i*256+255)] = i
	}
	for i := 0; i < 256; i++ {
		if smallSigned[int8(i)] != i || smallUnsigned[byte(i)] != i ||
			mediumSigned[int16(i*256+255)] != i || mediumUnsigned[uint16(i*256+255)] != i {
			panic("narrow key")
		}
	}
	var missing map[rune]int
	if value, ok := missing[-1]; ok || value != 0 {
		panic("nil map")
	}
	println("PASS")
}
