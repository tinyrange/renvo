package main

type CallbackStorage struct {
	kind   int
	first  *int
	second *int
	_      [0][]struct{}
}

var First = 17
var Second = 19
var Global = (CallbackStorage{kind: 7, first: &First, second: &Second})

func Identity(v CallbackStorage) CallbackStorage { return v }
func Read(v CallbackStorage) int                 { return v.kind + *v.first + *v.second }
func appMain() int {
	saved := Global
	if Read(saved) != 43 {
		return 1
	}
	Global = CallbackStorage{kind: 9, first: &Second, second: &First}
	if Read(Identity(Global)) != 45 || Read(saved) != 43 {
		return 2
	}
	if First != 17 || Second != 19 {
		return 3
	}
	print("PASS\n")
	return 0
}
