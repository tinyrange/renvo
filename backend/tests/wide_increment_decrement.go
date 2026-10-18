package main

type wideCounter struct{ value uint64 }

var wideGlobal uint64 = 0xffffffff
var evaluations int

func indexOnce() int            { evaluations++; return 0 }
func increment(v uint64) uint64 { v++; return v }
func decrement(v int64) int64   { v--; return v }
func appMain(args []string) int {
	if increment(0xffffffff) != 0x100000000 || decrement(0) != -1 {
		print("FAIL locals\n")
		return 1
	}
	wideGlobal++
	if wideGlobal != 0x100000000 {
		print("FAIL global increment\n")
		return 1
	}
	wideGlobal--
	if wideGlobal != 0xffffffff {
		print("FAIL global decrement\n")
		return 1
	}
	var c wideCounter
	c.value--
	if c.value != 0xffffffffffffffff {
		print("FAIL field decrement\n")
		return 1
	}
	c.value++
	if c.value != 0 {
		print("FAIL field increment\n")
		return 1
	}
	values := []uint64{0x100000000}
	values[indexOnce()]--
	if values[0] != 0xffffffff || evaluations != 1 {
		print("FAIL index\n")
		return 1
	}
	p := &values[0]
	(*p)++
	if values[0] != 0x100000000 {
		print("FAIL pointer\n")
		return 1
	}
	print("PASS\n")
	return 0
}
