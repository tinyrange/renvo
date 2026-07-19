package main

type globalNestedStructInner struct {
	count int
}

type globalNestedStructOuter struct {
	inner globalNestedStructInner
}

var globalNestedStructDefault = globalNestedStructOuter{inner: globalNestedStructInner{count: 4}}

func appMain() int {
	if globalNestedStructDefault.inner.count == 4 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
