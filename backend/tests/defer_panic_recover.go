package main

var deferOrder int

func deferFirst()  { deferOrder = deferOrder*10 + 1 }
func deferSecond() { deferOrder = deferOrder*10 + 2 }

func deferExplode() { panic("expected") }

func deferGuarded() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = deferOrder == 21
		}
	}()
	defer deferFirst()
	defer deferSecond()
	deferExplode()
	return false
}

func appMain() int {
	if deferGuarded() {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
