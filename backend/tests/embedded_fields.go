package main

type embeddedInner struct {
	X int
}

type embeddedMiddle struct {
	embeddedInner
}

type embeddedOuter struct {
	embeddedMiddle
}

type embeddedPointerOuter struct {
	*embeddedInner
}

type embeddedPointerMiddle struct {
	*embeddedInner
}

type embeddedNestedPointerOuter struct {
	embeddedPointerMiddle
}

func appMain() int {
	inner := embeddedInner{X: 41}
	value := embeddedOuter{embeddedMiddle{inner}}
	pointer := embeddedPointerOuter{&inner}
	nestedPointer := embeddedNestedPointerOuter{embeddedPointerMiddle{&inner}}
	if value.X != 41 || (embeddedOuter{embeddedMiddle{embeddedInner{X: 42}}}).X != 42 {
		return 1
	}
	value.X++
	pointer.X++
	nestedPointer.X++
	if value.embeddedMiddle.embeddedInner.X != 42 || inner.X != 43 || (embeddedPointerOuter{&inner}).X != 43 {
		return 1
	}
	print("PASS\n")
	return 0
}
