package main

type promotedBase struct{ Value int }
type promotedInner struct{ promotedBase }
type promotedOuter struct{ *promotedInner }

func appMain(args []string) int {
	w := promotedInner{promotedBase{42}}
	o := promotedOuter{&w}
	if o.Value != 42 || o.promotedBase.Value != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
