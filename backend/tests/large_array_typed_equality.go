package main

type Element struct {
	Tag  byte
	Text string
	_    int
	Data [17]byte
}
type Values [64]Element

func equal(a Values, b Values) bool { return a == b }
func appMain(args []string) int {
	var a, b Values
	for i := 0; i < len(a); i++ {
		a[i].Tag = byte(i)
		b[i].Tag = byte(i)
		a[i].Text = "same"
		b[i].Text = "x same"[2:]
		a[i].Data[16] = byte(63 - i)
		b[i].Data[16] = byte(63 - i)
	}
	if !equal(a, b) {
		print("FAIL equal\n")
		return 1
	}
	for _, index := range []int{0, 31, 63} {
		b[index].Data[16]++
		if equal(a, b) {
			print("FAIL nested mismatch\n")
			return 1
		}
		b[index].Data[16]--
		b[index].Text = "different"
		if equal(a, b) {
			print("FAIL string mismatch\n")
			return 1
		}
		b[index].Text = "same"
	}
	var x, y [65536]byte
	x[65535] = 7
	y[65535] = 7
	if x != y {
		print("FAIL large equal\n")
		return 1
	}
	y[65535] = 8
	if x == y {
		print("FAIL large mismatch\n")
		return 1
	}
	print("PASS\n")
	return 0
}
