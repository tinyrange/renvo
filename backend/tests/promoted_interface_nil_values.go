package main

type I interface{ method() int }
type ptr struct{}

func (v *ptr) method() int {
	if v == nil {
		return 9
	}
	return 1
}

type value struct{}

func (v value) method() int { return 2 }

type PW struct {
	pad int
	*ptr
}
type VW struct {
	pad int
	*value
}

func nilInterface() (ok bool) {
	defer func() { ok = recover() != nil }()
	var i I
	_ = i.method
	return false
}
func nilEmbedded() (ok bool) {
	defer func() { ok = recover() != nil }()
	var i I = VW{}
	i.method()
	return false
}

var recovered bool

type recovery struct{}

func (v *recovery) handle() { recovered = recover() != nil }
func deferred()             { var i interface{ handle() } = (*recovery)(nil); f := i.handle; defer f(); panic(7) }
func appMain(args []string) int {
	var i I = PW{}
	f := i.method
	if f() != 9 {
		return 1
	}
	if !nilInterface() {
		return 2
	}
	if !nilEmbedded() {
		return 3
	}
	deferred()
	if !recovered {
		return 4
	}
	print("PASS\n")
	return 0
}
