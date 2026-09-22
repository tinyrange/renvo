package main

type block struct{ bytes [259]byte }
type holder struct {
	before byte
	value  block
	after  byte
}

func copyBlock(dst *block, src *block) { *dst = *src }
func appMain() int {
	var src holder
	var dst holder
	src.before = 41
	src.after = 42
	dst.before = 51
	dst.after = 52
	for i := 0; i < 259; i++ {
		src.value.bytes[i] = byte(i)
	}
	copyBlock(&dst.value, &src.value)
	copyBlock(&dst.value, &dst.value)
	for i := 0; i < 259; i++ {
		if dst.value.bytes[i] != byte(i) {
			return 1
		}
	}
	if src.before != 41 || src.after != 42 || dst.before != 51 || dst.after != 52 {
		return 2
	}
	for n := 0; n <= 19; n++ {
		var bytes [24]byte
		for i := 0; i < 24; i++ {
			bytes[i] = byte(i)
		}
		copy(bytes[1:1+n], bytes[:n])
		for i := 0; i < n; i++ {
			if bytes[1+i] != byte(i) {
				return 3
			}
		}
		for i := 0; i < 24; i++ {
			bytes[i] = byte(i)
		}
		copy(bytes[:n], bytes[1:1+n])
		for i := 0; i < n; i++ {
			if bytes[i] != byte(i+1) {
				return 4
			}
		}
	}
	print("PASS\n")
	return 0
}
