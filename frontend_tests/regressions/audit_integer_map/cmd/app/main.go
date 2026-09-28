package main

func main() {
	m := make(map[int]int)
	for i := 0; i < 100000; i++ { m[i*17-50000] = i+1 }
	for i := 0; i < 100000; i++ { if m[i*17-50000] != i+1 { panic("lookup") } }
	for i := 0; i < 100000; i += 2 { delete(m,i*17-50000) }
	for i := 0; i < 100000; i++ {
		v, ok := m[i*17-50000]
		if i%2 == 0 { if ok { panic("deleted") } } else if !ok || v != i+1 { panic("moved entry") }
	}
	if len(m) != 50000 { panic("length") }
	for i := 0; i < 100000; i += 2 { m[i*17-50000] = -i }
	if len(m) != 100000 { panic("reinsertion") }
	clear(m)
	if len(m) != 0 { panic("clear") }
	m[0] = 42
	if m[0] != 42 { panic("reuse") }
	u := map[uint64]int{0: 1, 0xffffffffffffffff: 2, 0x8000000000000000: 3}
	delete(u, 0)
	if u[0xffffffffffffffff] != 2 || u[0x8000000000000000] != 3 { panic("wide keys") }
	println("PASS")
}
