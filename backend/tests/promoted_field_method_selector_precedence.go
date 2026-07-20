package main

type selectorPrecedenceBase struct {
	dismiss int
}

type selectorPrecedenceMenu struct {
	selectorPrecedenceBase
	open bool
}

func (menu *selectorPrecedenceMenu) dismiss() {
	menu.open = false
}

func appMain(args []string) int {
	menu := &selectorPrecedenceMenu{open: true}
	menu.dismiss()
	if menu.open {
		return 1
	}
	print("PASS\n")
	return 0
}
