package main

var forwardedRecoverValue interface{}

func forwardRecoverTarget() { forwardedRecoverValue = recover() }

// The frontend emits this marker for its typed deferred-call adapter.
//
//renvo:defer-forward call
func forwardRecoverCall() { forwardRecoverTarget() }

func appMain() int {
	func() { defer forwardRecoverCall(); panic(42) }()
	if forwardedRecoverValue == nil || forwardedRecoverValue.(int) != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
