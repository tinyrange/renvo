package widgets

type Event struct {
	X int
	Y int
}

type ClickHandler func(sender *Button, event Event)

type Button struct {
	Text  string
	Click ClickHandler
}

func (button *Button) Dispatch(event Event) {
	if button.Click != nil {
		button.Click(button, event)
	}
}
