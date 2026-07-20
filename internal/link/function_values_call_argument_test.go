package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestFunctionValueBoundMethodCallArgument(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/controls/controls.go", Src: []byte(`package controls

type Theme struct { Value int }
type ThemeHandler func(Theme)
type Control struct { handler ThemeHandler }

func (c *Control) SetThemeHandler(handler ThemeHandler) { c.handler = handler }
func (c *Control) Apply(theme Theme) {
	if c.handler != nil { c.handler(theme) }
}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/controls"

type widget struct { controls.Control; total int }
func (w *widget) apply(theme controls.Theme) { w.total += theme.Value }
func main() {
	var w widget
	w.SetThemeHandler(w.apply)
	w.Apply(controls.Theme{Value: 42})
	if w.total == 42 { print("PASS\n") }
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("receiver0: &w")) ||
		!bytes.Contains(linked.Program.Text, []byte("fn.receiver0.apply(")) {
		t.Fatalf("bound method call argument was not lowered:\n%s", linked.Program.Text)
	}
}
