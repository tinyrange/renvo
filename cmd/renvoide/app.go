package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func run(args []string, env []string) int {
	root := "."
	if len(args) > 1 && args[1] != "" {
		root = args[1]
	}
	window := graphics.NewWindow(graphics.WindowOptions{Title: "MiniIDE", Width: 1440, Height: 520})
	if window == nil {
		return 1
	}
	form := NewMainFormWithEnv(root, env)
	form.SetWindow(window)
	app := forms.NewApp(window, &form.Form)
	app.DispatchEvent = form.Dispatch
	app.AfterEvent = form.afterAppEvent
	return app.Run()
}

func (f *MainForm) afterAppEvent(event graphics.Event) {
	if f.takeEditorAnalysisTimer() {
		f.window.CancelTimer(editorAnalysisTimerID)
		f.window.SetTimer(editorAnalysisTimerID, 0.06)
	}
}
