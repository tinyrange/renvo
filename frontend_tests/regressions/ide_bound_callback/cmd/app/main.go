package main

import "example.com/renvotests/regressions/ide_bound_callback/widgets"

type MainForm struct {
	button widgets.Button
	total  int
}

func newMainForm() *MainForm {
	form := &MainForm{}
	form.button.Text = "Open"
	if form.button.Click != nil {
		return nil
	}
	form.button.Click = form.buttonClick
	return form
}

func (form *MainForm) buttonClick(sender *widgets.Button, event widgets.Event) {
	if sender.Text == "Open" {
		form.total += event.X + event.Y
	}
}

func main() {
	form := newMainForm()
	if form == nil || form.button.Click == nil {
		print("FAIL\n")
		return
	}
	form.button.Dispatch(widgets.Event{X: 19, Y: 23})
	form.button.Click = nil
	form.button.Dispatch(widgets.Event{X: 100, Y: 100})
	if form.total != 42 || form.button.Click != nil {
		print("FAIL\n")
		return
	}
	print("PASS\n")
	return
}
