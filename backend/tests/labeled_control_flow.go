package main

func functionEndingInLabel() {
	goto Done
Done:
}

func loopAfterEndingLabel() int {
	total := 0
	for i := 0; i < 4; i++ {
		total += i
	}
	return total
}

func appMain() int {
	functionEndingInLabel()
	ordinaryLoop := loopAfterEndingLabel()

	continued := 0
ContinueOuter:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			continued = continued*10 + i*3 + j + 1
			if j == 1 {
				continue ContinueOuter
			}
		}
	}

	broken := 0
	goto BreakOuter
BreakOuter:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			broken = broken*10 + i*3 + j + 1
			if j == 1 {
				break BreakOuter
			}
		}
	}

	if ordinaryLoop == 6 && continued == 124578 && broken == 12 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
