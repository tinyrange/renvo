package main

import "j5.nz/rtg/rtg/std/graphics"

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
	for {
		if form.Paint(window.Surface()) {
			if !window.Present() {
				window.Close()
				return 1
			}
		}
		event, ok := window.Wait()
		if !ok {
			window.Close()
			return 0
		}
		if event.Type == graphics.EventWindowClose {
			window.Close()
			return 0
		}
		form.Dispatch(event)
		if form.takeEditorAnalysisTimer() {
			window.CancelTimer(editorAnalysisTimerID)
			window.SetTimer(editorAnalysisTimerID, 0.06)
		}
	}
}
