package main

func pendingPure(n int) int {
	if n == 0 {
		return 7
	}
	return pendingPure(n - 1)
}
func pendingIndirect() bool { return recover() != nil }
func pendingDirect(seen *bool, stolen *bool, steps *int) {
	*steps = pendingPure(5)
	*stolen = pendingIndirect()
	*seen = recover() != nil
}

type pendingCatcher interface{ catch() }
type pendingReceiver struct {
	seen   *bool
	stolen *bool
	steps  *int
}

func (o pendingReceiver) catch() {
	*o.steps = pendingPure(4)
	*o.stolen = pendingIndirect()
	*o.seen = recover() != nil
}
func pendingCase(kind int) (seen bool, stolen bool, steps int) {
	o := pendingReceiver{&seen, &stolen, &steps}
	if kind == 0 {
		defer o.catch()
	}
	if kind == 1 {
		f := o.catch
		defer f()
	}
	if kind == 2 {
		var i pendingCatcher = o
		defer i.catch()
	}
	if kind == 3 {
		f := pendingDirect
		defer f(&seen, &stolen, &steps)
	}
	panic("owned")
}
func appMain() int {
	for kind := 0; kind < 4; kind++ {
		seen, stolen, steps := pendingCase(kind)
		if !seen || stolen || steps != 7 {
			panic("pending permission escaped or disappeared")
		}
	}
	if pendingIndirect() || pendingPure(3) != 7 {
		panic("ordinary pending entry")
	}
	print("PASS\n")
	return 0
}
