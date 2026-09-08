package main

type record struct {
	key   int
	value int
	next  *record
}

var initialized = [5]int{7, -11, 29, 3, 101}
var scratch [257]int
var visits int

type scalarWidths struct {
	a int8
	b uint8
	c int16
	d uint16
	e int32
	f uint32
}

func require(ok bool, name string) {
	if !ok {
		print("FAIL ", name, "\n")
		panic(name)
	}
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func eight(a, b, c, d, e, f, g, h int) int {
	return a + b*2 + c*3 + d*4 + e*5 + f*6 + g*7 + h*8
}

func incrementValue(n int) int              { return n*3 + 1 }
func applyValue(f func(int) int, n int) int { return f(n) }

func arithmetic() {
	for a := -37; a <= 37; a++ {
		for b := -11; b <= 11; b++ {
			if b != 0 {
				q, r := a/b, a%b
				require(q*b+r == a, "signed division identity")
				require(r == 0 || (r < 0) == (a < 0), "remainder sign")
			}
			require((a+b)-b == a && a*b == b*a, "arithmetic")
			require((a < b) == (b > a) && (a <= b) == !(a > b), "signed compare")
		}
	}
	x := uint32(0xf1234567)
	y := uint32(97)
	require(x/y*y+x%y == x, "unsigned division")
	require(x > y && x >= y && !(x < y), "unsigned compare")
	for s := uint(0); s < 31; s++ {
		require((uint32(1)<<s)>>s == 1, "variable shifts")
		require((-128>>s) < 0, "arithmetic shift")
	}
	require((x&255) == 103 && (x^x) == 0 && (x|255) == 0xf12345ff, "bitwise")
	dividends := []uint32{0, 1, 0x7fffffff, 0x80000000, 0xffffffff}
	divisors := []uint32{1, 2, 3, 97, 0x80000000, 0xffffffff}
	for _, numerator := range dividends {
		for _, divisor := range divisors {
			q, r := numerator/divisor, numerator%divisor
			require(q*divisor+r == numerator && r < divisor, "unsigned boundary division")
		}
	}
	for shift := uint(32); shift <= 65; shift++ {
		value := uint32(0xffffffff)
		value >>= shift
		require(value == 0 && uint32(0xffffffff)>>shift == 0, "oversized unsigned shift")
	}
	require(applyValue(incrementValue, 17) == 52, "indirect function call")
	require(fib(12) == 144, "recursion")
	require(eight(1, 2, 3, 4, 5, 6, 7, 8) == 204, "overflow arguments")
}

func crc(data []byte) uint32 {
	c := ^uint32(0)
	for _, b := range data {
		c ^= uint32(b)
		for bit := 0; bit < 8; bit++ {
			if c&1 != 0 {
				c = (c >> 1) ^ 0xedb88320
			} else {
				c >>= 1
			}
		}
	}
	return ^c
}

func algorithms() {
	var composite [512]bool
	count, sum := 0, 0
	for p := 2; p < len(composite); p++ {
		if !composite[p] {
			count++
			sum += p
			for n := p * p; n < len(composite); n += p {
				composite[n] = true
			}
		}
	}
	require(count == 97 && sum == 22548, "sieve")
	state := uint32(0x12345678)
	before := 0
	for i := range scratch {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		scratch[i] = int(state & 65535)
		before += scratch[i]
	}
	for i := 1; i < len(scratch); i++ {
		v, j := scratch[i], i
		for j > 0 && scratch[j-1] > v {
			scratch[j] = scratch[j-1]
			j--
		}
		scratch[j] = v
	}
	after := 0
	for i, v := range scratch {
		after += v
		if i > 0 {
			require(scratch[i-1] <= v, "sort order")
		}
	}
	require(before == after, "sort permutation")
	require(crc([]byte("123456789")) == 0xcbf43926, "CRC32")
}

func makeRecord(n int) record { return record{key: n, value: n * n} }
func mutate(r *record)        { r.value += r.key; visits++ }

func memory() {
	require(initialized[0]+initialized[1]+initialized[2]+initialized[3]+initialized[4] == 129, "initialized globals")
	var head *record
	for i := 0; i < 80; i++ {
		r := new(record)
		*r = makeRecord(i)
		r.next = head
		head = r
	}
	sum := 0
	for r := head; r != nil; r = r.next {
		mutate(r)
		sum += r.value
	}
	require(visits == 80 && sum == 170640, "heap linked list and aggregate return")
	data := make([]int, 0, 2)
	for i := 0; i < 100; i++ {
		data = append(data, i*i)
	}
	require(len(data) == 100 && data[99] == 9801, "append growth")
	alias := data[10:20]
	alias[3] = -7
	require(data[13] == -7, "slice alias")
	copy(data[1:10], data[:9])
	require(data[1] == 0 && data[9] == 64, "overlapping copy forward")
	copy(data[:9], data[1:10])
	require(data[8] == 64, "overlapping copy backward")
	var widths scalarWidths
	widths.a = -101
	widths.b = 241
	widths.c = -30001
	widths.d = 60001
	widths.e = -123456789
	widths.f = 0xfedcba98
	require(int(widths.a) == -101 && int(widths.b) == 241, "byte loads")
	require(int(widths.c) == -30001 && int(widths.d) == 60001, "halfword loads")
	require(widths.e == -123456789 && widths.f == 0xfedcba98, "word loads")
	text := "native" + " backend"
	require(len(text) == 14 && text[7:] == "backend", "strings")
}

func main() {
	arithmetic()
	algorithms()
	memory()
	print("PASS\n")
}
