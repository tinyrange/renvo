package main

var order int

func firstFloat() float64    { order = order*10 + 1; return 3.5 }
func secondFloat() float64   { order = order*10 + 2; return 4.5 }
func firstFloat32() float32  { order = order*10 + 1; return -3.5 }
func secondFloat32() float32 { order = order*10 + 2; return -4.5 }

func appMain(args []string) int {
	if firstFloat() >= secondFloat() || order != 12 {
		return 1
	}
	order = 0
	if firstFloat() == secondFloat() || order != 12 {
		return 2
	}
	order = 0
	if firstFloat32() <= secondFloat32() || order != 12 {
		return 3
	}
	order = 0
	if firstFloat32() != secondFloat32() {
		if order != 12 {
			return 4
		}
	} else {
		return 5
	}
	print("PASS\n")
	return 0
}
