package widget

func newWindow(options Options) *Window {
	if options.Title != "unused" || options.Hidden {
		return nil
	}
	return &Window{area: options.Width * options.Height}
}
