package main

type Color struct {
	R byte
	G byte
	B byte
	A byte
}

type Control struct {
	background Color
	invalid    bool
}

func newControl() *Control {
	return &Control{background: Color{R: 255, G: 255, B: 255, A: 255}}
}

func (control *Control) SetBackground(color Color) {
	if control == nil || control.background == color {
		return
	}
	control.background = color
	control.Invalidate()
}

func (control *Control) Invalidate() {
	if control != nil {
		control.invalid = true
	}
}

type ExplorerControl struct {
	Control
}

func main() {
	explorer := &ExplorerControl{}
	explorer.Control = *newControl()
	explorer.SetBackground(Color{R: 247, G: 248, B: 250, A: 255})
	if !explorer.invalid || explorer.background.R != 247 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
