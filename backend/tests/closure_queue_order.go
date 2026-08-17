package main

type closureQueueHolder struct {
	call func()
}

func closureQueueInvoke(holder *closureQueueHolder) { holder.call() }

func closureQueueNoop() {}

func closureQueueParent() {
	value := 1
	if false {
		func() { _ = value }()
	}
}

func appMain() int {
	var holder closureQueueHolder
	holder.call = closureQueueNoop
	closureQueueInvoke(&holder)
	closureQueueParent()
	print("PASS\n")
	return 0
}
