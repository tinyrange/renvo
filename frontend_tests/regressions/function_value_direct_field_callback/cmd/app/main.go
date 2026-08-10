package main

type button struct {
	click func()
}

var clicked bool

func click() {
	clicked = true
}

func main() {
	var button button
	button.click = click
	button.click()
	if clicked {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
