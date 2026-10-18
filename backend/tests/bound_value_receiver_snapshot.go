package main

type boundSnapshotValue struct{ N int }

func (v boundSnapshotValue) Read() int { return v.N }
func boundSnapshotNil() (ok bool) {
	defer func() { ok = recover() != nil }()
	var p *boundSnapshotValue
	_ = p.Read
	return false
}
func appMain() int {
	p := &boundSnapshotValue{N: 11}
	f := p.Read
	p.N = 19
	if f() != 11 || !boundSnapshotNil() {
		panic("method value must snapshot on binding")
	}
	print("PASS\n")
	return 0
}
