// Package omnibus is the OS-free core of the staged backend conformance
// artifact. The cmd/app package is only a hosted PASS adapter.
package omnibus

const (
	ProfileCore = 65537
	ResultSize  = 64

	stateRunning          = 1
	statePassed           = 2
	stateFailedComparison = 3
)

const (
	offsetMagic           = 0
	offsetVersion         = 4
	offsetSize            = 6
	offsetProfile         = 8
	offsetState           = 12
	offsetCurrentProbe    = 16
	offsetFailureProbe    = 20
	offsetCompletedProbes = 24
	offsetExpected        = 32
	offsetObserved        = 40
	offsetSignature       = 48
	offsetSequence        = 56
)

var renvores [ResultSize]byte
var signatureLo int32
var signatureHi int32
var completed int32
var sequence int32
var state int32

var globalCounter int
var globalPair pair

type pair struct {
	x int
	y int
}

func (p *pair) weighted(scale int) int {
	return p.x*scale + p.y
}

func ResultByte(index int) byte {
	return renvores[index]
}

func Passed(expectedLo int32, expectedHi int32, expectedCount int) bool {
	return state == statePassed && signatureLo == expectedLo && signatureHi == expectedHi && int(completed) == expectedCount
}

func Stage0() bool {
	reset(ProfileCore)
	if !probeStage0() {
		return false
	}
	pass()
	return true
}

func Stage1() bool {
	reset(ProfileCore)
	if !probeStage0() || !probeStage1() {
		return false
	}
	pass()
	return true
}

func Stage2() bool {
	reset(ProfileCore)
	if !probeStage0() || !probeStage1() || !probeStage2() {
		return false
	}
	pass()
	return true
}

func Stage3() bool {
	reset(ProfileCore)
	if !probeStage0() || !probeStage1() || !probeStage2() || !probeStage3() {
		return false
	}
	pass()
	return true
}

func RunAll() bool {
	return Stage3()
}

func probeStage0() bool {
	return check(1, 37, constantReturn())
}

func probeStage1() bool {
	arithmetic := (13*7 - 5) / 2
	if !check(101, 43, arithmetic) {
		return false
	}

	control := 0
	for i := 0; i < 8; i++ {
		if i%2 == 0 {
			control += i * 3
		} else {
			control -= i
		}
	}
	if !check(102, 20, control) {
		return false
	}

	bits := ((5 << 4) | (48 >> 2)) ^ 3
	return check(103, 95, bits)
}

func probeStage2() bool {
	if !check(201, 204, manyArgs(1, 2, 3, 4, 5, 6, 7, 8)) {
		return false
	}
	if !check(202, 720, factorial(6)) {
		return false
	}
	return check(203, 41, stackMix(2, 3, 4))
}

func probeStage3() bool {
	globalCounter = 0
	globalCounter += 29
	if !check(301, 29, globalCounter) {
		return false
	}

	globalPair = pair{x: 7, y: 11}
	p := &globalPair
	p.y = 13
	if !check(302, 20, p.x+p.y) {
		return false
	}

	values := arrayValue(4)
	if !check(303, 27, values[0]+values[1]+values[2]) {
		return false
	}

	local := pair{x: 2, y: 5}
	return check(304, 11, local.weighted(3))
}

func constantReturn() int {
	return 37
}

func manyArgs(a int, b int, c int, d int, e int, f int, g int, h int) int {
	return a + b*2 + c*3 + d*4 + e*5 + f*6 + g*7 + h*8
}

func factorial(value int) int {
	if value <= 1 {
		return 1
	}
	return value * factorial(value-1)
}

func stackMix(a int, b int, c int) int {
	d := a + b
	e := b + c
	return d*e + a + c
}

func arrayValue(base int) [3]int {
	return [3]int{base, base + 5, base + 10}
}

func check(id int32, expected int, observed int) bool {
	beginProbe(id)
	if observed != expected {
		failComparison(id, int32(expected), int32(observed))
		return false
	}
	completeProbe(id, int32(observed))
	return true
}

func reset(profile int32) {
	signatureLo = 0
	signatureHi = 0
	completed = 0
	sequence = 0
	state = 0
	put32(offsetMagic, 1380406354)
	put16(offsetVersion, 1)
	put16(offsetSize, ResultSize)
	put32(offsetProfile, profile)
	put32(offsetState, state)
}

func beginProbe(id int32) {
	put32(offsetCurrentProbe, id)
	bumpSequence()
	state = stateRunning
	put32(offsetState, state)
}

func completeProbe(id int32, observed int32) {
	put32(offsetCurrentProbe, id)
	completed++
	put32(offsetCompletedProbes, completed)
	signatureLo = (signatureLo ^ id ^ observed) * 16777619
	signatureHi = (signatureHi ^ id) * -2048144777
	put32(offsetSignature, signatureLo)
	put32(offsetSignature+4, signatureHi)
	bumpSequence()
}

func failComparison(id int32, expected int32, observed int32) {
	put32(offsetCurrentProbe, id)
	put32(offsetFailureProbe, id)
	put32(offsetExpected, expected)
	put32(offsetExpected+4, 0)
	put32(offsetObserved, observed)
	put32(offsetObserved+4, 0)
	bumpSequence()
	state = stateFailedComparison
	put32(offsetState, state)
}

func pass() {
	put32(offsetSignature, signatureLo)
	put32(offsetSignature+4, signatureHi)
	bumpSequence()
	state = statePassed
	put32(offsetState, state)
}

func bumpSequence() {
	sequence++
	put32(offsetSequence, sequence)
}

func put16(offset int, value int) {
	renvores[offset] = byte(value)
	renvores[offset+1] = byte(value >> 8)
}

func put32(offset int, value int32) {
	renvores[offset] = byte(value)
	renvores[offset+1] = byte(value >> 8)
	renvores[offset+2] = byte(value >> 16)
	renvores[offset+3] = byte(value >> 24)
}
