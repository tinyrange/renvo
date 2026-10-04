package main

type CallbackEmbeddedBase struct{ Value int }
type CallbackEmbeddedAlias = CallbackEmbeddedBase
type CallbackEmbeddedArg = struct{ CallbackEmbeddedBase }
type CallbackEmbeddedOther = struct{ CallbackEmbeddedBase }
type CallbackEmbeddedDifferent = struct{ CallbackEmbeddedAlias }

func embeddedApply(f func(CallbackEmbeddedArg) int) int {
	return f(CallbackEmbeddedArg{CallbackEmbeddedBase{42}})
}
func embeddedVisit(v CallbackEmbeddedOther) int { return v.CallbackEmbeddedBase.Value }
func appMain() int {
	var f func(func(CallbackEmbeddedOther) int) int = embeddedApply
	if f(embeddedVisit) != 42 {
		return 1
	}
	var boxed any = CallbackEmbeddedArg{CallbackEmbeddedBase{43}}
	v, ok := boxed.(CallbackEmbeddedOther)
	if !ok || v.CallbackEmbeddedBase.Value != 43 {
		return 2
	}
	if _, ok := boxed.(CallbackEmbeddedDifferent); ok {
		return 3
	}
	print("PASS\n")
	return 0
}
