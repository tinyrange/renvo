package main

type functionValueDirectFieldHolder struct {
	callback func()
}

var functionValueDirectFieldState int

func functionValueDirectFieldTarget() {
	functionValueDirectFieldState = 42
}

func functionValueDirectFieldInstall(holder *functionValueDirectFieldHolder, callback func()) {
	holder.callback = callback
}

func functionValueDirectFieldInvoke(holder *functionValueDirectFieldHolder) {
	holder.callback()
}

func appMain(args []string) int {
	var holder functionValueDirectFieldHolder
	functionValueDirectFieldInstall(&holder, functionValueDirectFieldTarget)
	functionValueDirectFieldInvoke(&holder)
	if functionValueDirectFieldState != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
