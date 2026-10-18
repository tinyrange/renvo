package main

type Wide uint64
type Alias = Wide
type Signed int64
type Native int
type NativeUnsigned uint
type Address uintptr

func keyAt(i int) uint64 {
	n := uint64(i%64 + 1)
	switch i / 64 {
	case 0:
		return (n << 32) | 0x5a5a
	case 1:
		return (uint64(0x87654321) << 32) | n
	case 2:
		return (n << 32) | n
	default:
		return (uint64(0x80000000)+n)<<32 | (0xffffffff - n)
	}
}

func main() {
	wide := make(map[Alias]int)
	signed := make(map[Signed]int)
	alias := wide
	for i := 0; i < 256; i++ {
		key := keyAt(i)
		wide[Wide(key)] = i + 1
		signed[Signed(int64(key))] = i + 1
	}
	if len(wide) != 256 || len(signed) != 256 {
		panic("full-width identity")
	}
	for i := 0; i < 256; i += 3 {
		delete(alias, Wide(keyAt(i)))
		delete(signed, Signed(int64(keyAt(i))))
	}
	for i := 0; i < 256; i++ {
		key := keyAt(i)
		u, uok := wide[Wide(key)]
		s, sok := signed[Signed(int64(key))]
		if i%3 == 0 {
			if uok || sok || u != 0 || s != 0 {
				panic("deleted")
			}
		} else if !uok || !sok || u != i+1 || s != i+1 {
			panic("collision chain")
		}
	}
	for i := 0; i < 256; i += 3 {
		alias[Wide(keyAt(i))] = i + 1
		signed[Signed(int64(keyAt(i)))] = i + 1
	}
	count := 0
	for key, value := range alias {
		if signed[Signed(int64(key))] != value {
			panic("range")
		}
		count++
	}
	if count != 256 {
		panic("reinsertion")
	}
	clear(alias)
	clear(signed)
	wide[Wide(0xffffffffffffffff)] = 0
	signed[Signed(-9223372036854775807-1)] = 17
	if value, ok := alias[Wide(0xffffffffffffffff)]; !ok || value != 0 {
		panic("present zero")
	}
	if signed[Signed(-9223372036854775807-1)] != 17 || len(alias) != 1 {
		panic("boundary reuse")
	}

	// On 64-bit targets these keys differ in both halves; on 32-bit targets
	// the conversions deliberately retain only the native-width bit pattern.
	native := make(map[Native]int)
	unsigned := make(map[NativeUnsigned]int)
	addresses := make(map[Address]int)
	for i := 0; i < 128; i++ {
		key := (uint64(i+1) << 32) | uint64(i*0x100000+0xfffff)
		native[Native(int(key))] = i + 1
		unsigned[NativeUnsigned(uint(key))] = i + 1
		addresses[Address(uintptr(key))] = i + 1
	}
	for i := 0; i < 128; i++ {
		key := (uint64(i+1) << 32) | uint64(i*0x100000+0xfffff)
		if native[Native(int(key))] != i+1 || unsigned[NativeUnsigned(uint(key))] != i+1 || addresses[Address(uintptr(key))] != i+1 {
			panic("native width")
		}
	}
	native[-1] = 0
	if value, ok := native[-1]; !ok || value != 0 {
		panic("native sign")
	}
	var missing map[Signed]int
	delete(missing, -1)
	if value, ok := missing[-1]; ok || value != 0 {
		panic("nil")
	}
	println("PASS")
}
