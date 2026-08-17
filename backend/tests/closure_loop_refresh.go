package main

func setClosureLoopFlag(flag *bool) { *flag = true }

func appMain() int {
	ready := false
	run := func() {
		for !ready {
			setClosureLoopFlag(&ready)
		}
	}
	run()
	if ready {
		print("PASS\n")
	}
	return 0
}
