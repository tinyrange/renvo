package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestMenuBarShortcutRunsBeforeFocusedControl(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	box := NewTextBox()
	box.SetBounds(graphics.R(10, 50, 240, 30))
	box.SetFont(graphics.NewBuiltinFont(1))
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 220, 36))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	save := NewMenuItem("Save")
	save.SetShortcut(Shortcut{Key: graphics.KeyS, Primary: true, Text: "Ctrl/Cmd+S"})
	called := 0
	save.Activate = func() { called++ }
	file.Add(save)
	menu.Add(file)
	form.Add(&box.Control)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)
	box.Focus()

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyS, Modifiers: graphics.ModifierControl})
	if called != 1 || !box.Focused() {
		t.Fatalf("shortcut state = called %d focused %v", called, box.Focused())
	}
	save.SetEnabled(false)
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyS, Modifiers: graphics.ModifierCommand})
	if called != 1 {
		t.Fatalf("disabled shortcut called %d times", called)
	}
}

func TestMenuBarOpensSelectsAndDismissesWithoutOwningFocus(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(10, 5, 220, 34))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	open := NewMenuItem("Open...")
	called := 0
	open.Activate = func() { called++ }
	file.Add(open)
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)

	closed := menu.Bounds()
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 25, Y: 20})
	if !menu.Open() || menu.Bounds().Height() <= closed.Height() {
		t.Fatalf("menu did not expand: open %v bounds %#v", menu.Open(), menu.Bounds())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 30, Y: 55})
	if called != 1 || menu.Open() || menu.Bounds() != closed {
		t.Fatalf("selection state = called %d open %v bounds %#v", called, menu.Open(), menu.Bounds())
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 25, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 300, Y: 150})
	if menu.Open() {
		t.Fatal("outside click did not dismiss menu")
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 25, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: 300, Y: 150})
	if menu.Open() {
		t.Fatal("moving away did not dismiss menu")
	}
}

func TestMenuBarKeyboardNavigationActivatesSelectedItem(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	menu := NewMenuBar()
	menu.SetAccessibilityID("main-menu")
	menu.SetBounds(graphics.R(0, 0, 220, 34))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	file.Add(NewMenuItem("Open"))
	save := NewMenuItem("Save")
	called := 0
	save.Activate = func() { called++ }
	file.Add(save)
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 15, Y: 15})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnter})
	if called != 1 || menu.Open() {
		t.Fatalf("keyboard selection = called %d open %v", called, menu.Open())
	}
	driver := NewAutomationDriver(&form)
	if !driver.Invoke("main-menu-menu-1") || !menu.Open() {
		t.Fatal("semantic menu invocation did not open File")
	}
	if !driver.Invoke("main-menu-menu-1-item-2") || called != 2 || menu.Open() {
		t.Fatalf("semantic item state = called %d open %v", called, menu.Open())
	}
}

func TestFullWidthMenuBarDismissesAcrossItsExpandedHitRectangle(t *testing.T) {
	var form Form
	form.Initialize(900, 500)
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 900, 34))
	file := NewMenu("File")
	file.Add(NewMenuItem("Open"))
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 17})
	if !menu.Open() {
		t.Fatal("menu did not open")
	}
	// This point shares the popup's row but is horizontally outside its drop.
	// A full-width menu control still owns the rectangular hit test here.
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: 700, Y: 55})
	if menu.Open() {
		t.Fatal("horizontal pointer departure did not dismiss full-width menu")
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 17})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 700, Y: 55})
	if menu.Open() {
		t.Fatal("horizontal outside click did not dismiss full-width menu")
	}
}

func TestMenuBarDismissesWhenApplicationLosesActivation(t *testing.T) {
	var form Form
	form.Initialize(640, 400)
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 640, 34))
	file := NewMenu("File")
	file.Add(NewMenuItem("Open"))
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)
	closed := menu.Bounds()

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 17})
	if !menu.Open() {
		t.Fatal("menu did not open")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventWindowFocusLost})
	if menu.Open() || menu.Control.popup || menu.Bounds() != closed {
		t.Fatalf("deactivated menu state = open %v popup %v bounds %#v", menu.Open(), menu.Control.popup, menu.Bounds())
	}
}

func TestMenuBarStandardKeyboardDismissalAndSelection(t *testing.T) {
	var form Form
	form.Initialize(640, 400)
	first := NewTextBox()
	first.SetBounds(graphics.R(10, 60, 100, 30))
	second := NewButton()
	second.SetBounds(graphics.R(120, 60, 100, 30))
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 640, 34))
	file := NewMenu("File")
	file.Add(NewMenuItem("Open"))
	file.Add(NewMenuItem("Save"))
	menu.Add(file)
	form.Add(&first.Control)
	form.Add(&second.Control)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)
	first.Focus()

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 17})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "s"})
	if menu.selectedItem != 1 || first.Text() != "" {
		t.Fatalf("menu type-ahead = selected %d, focused text %q", menu.selectedItem, first.Text())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnd})
	if menu.selectedItem != 1 {
		t.Fatalf("End selected item %d, want 1", menu.selectedItem)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyHome})
	if menu.selectedItem != 0 {
		t.Fatalf("Home selected item %d, want 0", menu.selectedItem)
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyTab})
	if menu.Open() || !second.Focused() {
		t.Fatalf("Tab state = menu open %v, second focused %v", menu.Open(), second.Focused())
	}
}

func TestDockedMenuBarKeepsItsBarHeightWhilePopupIsOpen(t *testing.T) {
	var form Form
	form.Initialize(400, 240)
	bar := NewMenuBar()
	bar.SetBounds(graphics.R(0, 0, 120, 34))
	bar.SetDock(DockTop)
	menu := NewMenu("File")
	menu.Add(NewMenuItem("Open"))
	bar.Add(menu)
	form.SetMenuBar(bar)
	form.Add(&bar.Control)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 10, Y: 10})
	if !bar.Open() || bar.Control.Bounds().Height() <= 34 {
		t.Fatalf("menu did not expand: open %v bounds %#v", bar.Open(), bar.Control.Bounds())
	}
	form.SetClientSize(520, 260)
	if bar.barBounds != graphics.R(0, 0, 520, 34) {
		t.Fatalf("resized dock bar = %#v", bar.barBounds)
	}
	bar.dismiss()
	if bar.Control.Bounds() != graphics.R(0, 0, 520, 34) {
		t.Fatalf("dismissed dock bar = %#v", bar.Control.Bounds())
	}
}
