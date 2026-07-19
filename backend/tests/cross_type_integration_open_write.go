package main

type namedInt int
type namedByte byte
type mixRecord struct {
	n    int
	b    byte
	ok   bool
	text string
	data []byte
}

var globalTarget int = 4
var globalPointer *int = &globalTarget

const crossLimit = 3

func namedAdd(v namedInt) int { return int(v) + 5 }
func namedB() namedByte       { return namedByte('q') }

func offsetLen() int64 {
	var buf []byte
	buf = append(buf, 'a')
	buf = append(buf, 'b')
	return int64(len(buf)) + int64(5)
}
func boolArith(r mixRecord) int {
	if r.ok {
		return r.n * 3
	}
	return r.n
}
func copyRec(s string, i int, out []byte) int {
	if i >= len(s) {
		return 0
	}
	out = append(out, s[i])
	return int(out[len(out)-1]) + copyRec(s, i+1, out)
}
func offsetLoop(n int) int64 {
	var off int64
	i := 0
	for i < n {
		off += int64(i * 2)
		i += 1
	}
	return off
}
func addByte(r mixRecord, b byte) mixRecord { r.data = append(r.data, b); return r }
func makeNamed(n namedInt) mixRecord        { return mixRecord{n: int(n), b: 'n'} }
func acceptBoth(data []byte, p *int) int    { return int(data[0]) + *p }

func appMain(args []string) int {
	var buf []byte
	buf = append(buf, "A"[0])
	buf = append(buf, byte(66))
	fd := open("renvo_973_cross.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0973 open write failed\n")
		return 1
	}
	if write(fd, buf, int64(0)) != 2 {
		print("RENVO-0973 file write failed\n")
		close(fd)
		return 2
	}
	close(fd)
	print("PASS\n")
	return 0
}
