package main

type Handler func()

type Control struct {
	active  bool
	dismiss Handler
}

type Menu struct {
	Control
	open bool
}

func (menu *Menu) dismiss() {
	menu.open = false
}

type Form struct {
	controls []*Control
}

func (form *Form) dismissActive() {
	for i := 0; i < len(form.controls); i++ {
		control := form.controls[i]
		if control.active && control.dismiss != nil {
			control.dismiss()
		}
	}
}

func main() {
	menu := &Menu{open: true}
	menu.active = true
	menu.Control.dismiss = menu.dismiss
	form := &Form{controls: []*Control{&menu.Control}}
	form.dismissActive()
	if menu.open {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
