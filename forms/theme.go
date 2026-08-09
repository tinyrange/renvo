package forms

import "renvo.dev/std/graphics"

// Theme is the complete color contract used by Forms controls. Applications
// can start with LightTheme or DarkTheme, adjust individual fields, then apply
// the result to a form at runtime.
type Theme struct {
	Window        graphics.Color
	Surface       graphics.Color
	SurfaceRaised graphics.Color
	Field         graphics.Color
	Text          graphics.Color
	MutedText     graphics.Color
	Border        graphics.Color
	Hover         graphics.Color
	Selection     graphics.Color
	Accent        graphics.Color
	AccentText    graphics.Color
	Disabled      graphics.Color
}

func LightTheme() Theme {
	return Theme{
		Window:        graphics.RGBA(245, 247, 250, 255),
		Surface:       graphics.RGBA(255, 255, 255, 255),
		SurfaceRaised: graphics.RGBA(235, 239, 244, 255),
		Field:         graphics.RGBA(249, 250, 252, 255),
		Text:          graphics.RGBA(30, 35, 43, 255),
		MutedText:     graphics.RGBA(82, 91, 103, 255),
		Border:        graphics.RGBA(190, 198, 209, 255),
		Hover:         graphics.RGBA(229, 239, 251, 255),
		Selection:     graphics.RGBA(207, 228, 252, 255),
		Accent:        graphics.RGBA(25, 118, 210, 255),
		AccentText:    graphics.RGBA(255, 255, 255, 255),
		Disabled:      graphics.RGBA(128, 137, 149, 255),
	}
}

func DarkTheme() Theme {
	return Theme{
		Window:        graphics.RGBA(27, 31, 38, 255),
		Surface:       graphics.RGBA(35, 40, 49, 255),
		SurfaceRaised: graphics.RGBA(45, 51, 62, 255),
		Field:         graphics.RGBA(29, 34, 42, 255),
		Text:          graphics.RGBA(232, 235, 240, 255),
		MutedText:     graphics.RGBA(164, 172, 184, 255),
		Border:        graphics.RGBA(76, 84, 98, 255),
		Hover:         graphics.RGBA(48, 61, 78, 255),
		Selection:     graphics.RGBA(39, 73, 108, 255),
		Accent:        graphics.RGBA(65, 145, 230, 255),
		AccentText:    graphics.RGBA(255, 255, 255, 255),
		Disabled:      graphics.RGBA(108, 116, 130, 255),
	}
}

func (f *Form) Theme() Theme {
	if f == nil {
		return LightTheme()
	}
	return f.theme
}

func (f *Form) ApplyTheme(theme Theme) {
	if f == nil {
		return
	}
	f.theme = theme
	f.themeApplied = true
	f.background = theme.Window
	for i := 0; i < len(f.controls); i++ {
		f.controls[i].ApplyTheme(theme)
	}
	f.Invalidate(f.clientRect())
}

func controlTheme(control *Control) Theme {
	if control != nil && control.form != nil {
		return control.form.theme
	}
	if control != nil && control.hasTheme {
		return control.theme
	}
	return LightTheme()
}

func controlForeground(control *Control) graphics.Color {
	theme := controlTheme(control)
	if control != nil && !control.Enabled() {
		return theme.Disabled
	}
	if control == nil {
		return theme.Text
	}
	return control.Foreground()
}

func controlAccent(control *Control) graphics.Color {
	theme := controlTheme(control)
	if control != nil && !control.Enabled() {
		return theme.Disabled
	}
	return theme.Accent
}

func applyFieldTheme(control *Control, theme Theme) {
	control.SetBackground(theme.Field)
	control.SetForeground(theme.Text)
}

func applySurfaceTheme(control *Control, theme Theme) {
	control.SetBackground(theme.Surface)
	control.SetForeground(theme.Text)
}

func applyRaisedTheme(control *Control, theme Theme) {
	control.SetBackground(theme.SurfaceRaised)
	control.SetForeground(theme.Text)
}

func applyTransparentTheme(control *Control, theme Theme) {
	control.SetBackground(graphics.RGBA(theme.Window.R, theme.Window.G, theme.Window.B, 0))
	control.SetForeground(theme.Text)
}
