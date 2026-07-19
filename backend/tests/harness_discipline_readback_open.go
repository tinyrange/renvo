package main

type checkRecord struct {
	value int
	ok    bool
	data  []byte
}

var harnessSeed int = 9

const harnessLimit = 4

func unusualFold(v int) int { return v*2 + 1 }
func recursiveMark(n int, p *int) {
	if n == 0 {
		return
	}
	*p += n
	recursiveMark(n-1, p)
}

func hostSensitive() int { value := len("abc"); return value * 7 }

func appMain(args []string) int {
	fd := open("renvo_982_harness.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0982 readback open failed\n")
		return 1
	}
	var w []byte
	w = append(w, 'o')
	w = append(w, 'k')
	write(fd, w, int64(0))
	var r []byte
	r = append(r, byte(0))
	r = append(r, byte(0))
	if read(fd, r, int64(0)) != 2 {
		print("RENVO-0982 readback read failed\n")
		close(fd)
		return 2
	}
	if r[0] != 'o' || r[1] != 'k' {
		print("RENVO-0982 readback data failed\n")
		close(fd)
		return 3
	}
	close(fd)
	print("PASS\n")
	return 0
}
