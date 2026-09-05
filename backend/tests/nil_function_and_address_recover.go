package main

func nilFunction() (ok bool) {
	defer func() { ok = recover() != nil }()
	var f func()
	f()
	return
}

func deferredNilFunction() (ok bool) {
	defer func() { ok = recover() != nil }()
	var f func()
	defer f()
	return
}

func nilAddress() (ok bool) {
	defer func() { ok = recover() != nil }()
	var p *int
	_ = &*p
	return
}

func appMain(args []string) int {
	if !nilFunction() {
		return 1
	}
	if !deferredNilFunction() {
		return 2
	}
	if !nilAddress() {
		return 3
	}
	x := 4
	p := &x
	if &*p != p {
		return 4
	}
	print("PASS\n")
	return 0
}
