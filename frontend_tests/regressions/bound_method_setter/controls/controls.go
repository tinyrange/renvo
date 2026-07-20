package controls

type Theme struct {
	Value int
}

type ThemeHandler func(theme Theme)

type Control struct {
	handler ThemeHandler
}

func (c *Control) SetThemeHandler(handler ThemeHandler) {
	if handler == nil {
		c.handler = nil
	} else {
		c.handler = handler
	}
}

func (c *Control) Apply(theme Theme) {
	if c.handler != nil {
		c.handler(theme)
	}
}
