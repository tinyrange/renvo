package main

type functionValueNoTargetHandler func()

type functionValueNoTargetItem struct {
	activate functionValueNoTargetHandler
}

func functionValueNoTargetInvoke(item *functionValueNoTargetItem) {
	if item.activate != nil {
		item.activate()
	}
}

func appMain(args []string) int {
	functionValueNoTargetInvoke(&functionValueNoTargetItem{})
	print("PASS\n")
	return 0
}
