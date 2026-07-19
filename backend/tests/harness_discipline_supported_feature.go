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
	var data []byte
	data = append(data, byte(1))
	data = append(data, byte(2))
	if len(data) != 2 || data[0]+data[1] != byte(3) {
		print("RENVO-0992 supported feature failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
