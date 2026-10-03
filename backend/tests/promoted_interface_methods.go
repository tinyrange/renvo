package main

type promotedHidden interface{ boxed() int }
type promotedValue int

func (v promotedValue) boxed() int { return int(v) }

type promotedOther int

func (v promotedOther) hidden() int { return int(v) }

type promotedMixed struct {
	padding int
	promotedHidden
	promotedOther
}

type promotedNested struct {
	padding int
	*promotedMixed
}

func appMain(args []string) int {
	mixed := promotedMixed{11, promotedValue(42), 7}
	var both interface {
		promotedHidden
		hidden() int
	} = mixed
	if both.boxed() != 42 || both.hidden() != 7 {
		return 1
	}
	if mixed.boxed() != 42 || mixed.hidden() != 7 {
		return 2
	}
	valueMethod := mixed.hidden
	mixed.promotedOther = 9
	if valueMethod() != 7 {
		return 3
	}
	var dynamic interface{} = mixed
	view, ok := dynamic.(interface {
		boxed() int
		hidden() int
	})
	if !ok || view.boxed() != 42 || view.hidden() != 9 {
		return 4
	}
	nested := &promotedNested{13, &mixed}
	var hidden promotedHidden = nested
	bound := hidden.boxed
	mixed.promotedHidden = promotedValue(51)
	if hidden.boxed() != 51 || bound() != 51 || nested.boxed() != 51 {
		return 5
	}
	mixed.promotedHidden = promotedMixed{17, promotedValue(67), 19}
	if bound() != 67 {
		return 6
	}
	print("PASS\n")
	return 0
}
