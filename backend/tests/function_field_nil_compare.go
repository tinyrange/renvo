package main

type callbackHolder struct {
	callback func()
}

func appMain(args []string) int {
	var holder callbackHolder
	hasCallback := holder.callback != nil
	if hasCallback {
		return 1
	}
	print("PASS\n")
	return 0
}
