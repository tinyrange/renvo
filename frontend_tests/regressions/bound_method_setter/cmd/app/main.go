package main

import "example.com/renvotests/regressions/bound_method_setter/controls"

type widget struct {
	controls.Control
	total int
}

func (w *widget) apply(theme controls.Theme) {
	w.total += theme.Value
}

func main() {
	var w widget
	w.SetThemeHandler(w.apply)
	w.Apply(controls.Theme{Value: 42})
	if w.total != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
