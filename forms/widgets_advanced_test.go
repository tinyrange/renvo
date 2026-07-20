package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestAdvancedControlsExposePropertiesInteractionAndSemantics(t *testing.T) {
	font := graphics.NewBuiltinFont(1)
	var form Form
	form.Initialize(900, 620)

	combo := NewComboBox()
	combo.SetAccessibilityID("combo")
	combo.SetBounds(graphics.R(10, 10, 180, 32))
	combo.SetFont(font)
	combo.AddItem("Debug")
	combo.AddItem("Release")
	combo.SetSelectedIndex(0)

	list := NewListBox()
	list.SetAccessibilityID("list")
	list.SetBounds(graphics.R(10, 55, 180, 80))
	list.SetFont(font)
	list.AddItem("One")
	list.AddItem("Two")

	table := NewListView()
	table.SetAccessibilityID("table")
	table.SetBounds(graphics.R(210, 10, 260, 125))
	table.SetFont(font)
	table.AddColumn("Name")
	table.AddColumn("Value")
	table.AddRow([]string{"Target", "browser/wasm32"})

	tree := NewTreeView()
	tree.SetAccessibilityID("tree")
	tree.SetBounds(graphics.R(490, 10, 180, 125))
	tree.SetFont(font)
	tree.AddNode("Project", 0)
	tree.AddNode("main.go", 1)

	tabs := NewTabControl()
	tabs.SetAccessibilityID("tabs")
	tabs.SetBounds(graphics.R(10, 150, 300, 40))
	tabs.SetFont(font)
	tabs.AddTab("Code")
	tabs.AddTab("Designer")

	progress := NewProgressBar()
	progress.SetBounds(graphics.R(10, 210, 260, 22))
	progress.SetValue(65)

	number := NewNumericUpDown()
	number.SetBounds(graphics.R(290, 205, 120, 32))
	number.SetFont(font)
	number.SetRange(1, 10)
	number.SetValue(4)

	slider := NewSlider()
	slider.SetBounds(graphics.R(430, 205, 240, 32))
	slider.SetValue(25)

	group := NewGroupBox()
	group.SetBounds(graphics.R(10, 255, 260, 100))
	group.SetFont(font)
	group.SetText("Options")

	split := NewSplitContainer()
	split.SetBounds(graphics.R(290, 255, 380, 100))
	split.SetSplitterDistance(140)

	toolbar := NewToolBar()
	toolbar.SetAccessibilityID("toolbar")
	toolbar.SetBounds(graphics.R(10, 375, 660, 36))
	toolbar.SetFont(font)
	clicked := 0
	toolbar.AddButton("Save", func() { clicked++ })

	status := NewStatusBar()
	status.SetBounds(graphics.R(10, 430, 660, 28))
	status.SetFont(font)
	status.SetText("Ready")

	controls := []*Control{&combo.Control, &list.Control, &table.Control, &tree.Control, &tabs.Control, &progress.Control, &number.Control, &slider.Control, &group.Control, &split.Control, &toolbar.Control, &status.Control}
	for i := 0; i < len(controls); i++ {
		form.Add(controls[i])
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 30, Y: 90, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 30, Y: 90, Button: 1})
	if list.SelectedIndex() != 1 || list.SelectedItem() != "Two" {
		t.Fatalf("list selection = %d, %q", list.SelectedIndex(), list.SelectedItem())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: 650, Y: 220})
	if slider.Value() != 25 {
		t.Fatalf("slider changed from hover alone: %d", slider.Value())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 650, Y: 220, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 650, Y: 220, Button: 1})
	if slider.Value() < 80 {
		t.Fatalf("slider pointer value = %d", slider.Value())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 430, Y: 300, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: 480, Y: 300, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 480, Y: 300, Button: 1})
	if split.SplitterDistance() != 190 {
		t.Fatalf("dragged splitter distance = %d, want 190", split.SplitterDistance())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 35, Y: 392, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 35, Y: 392, Button: 1})
	if clicked != 1 {
		t.Fatalf("toolbar clicks = %d", clicked)
	}

	surface := graphics.NewSurface(900, 620)
	if !form.Paint(surface) {
		t.Fatal("advanced control form did not paint")
	}
	nodes := form.AccessibilitySnapshot()
	for _, id := range []string{"combo-item-1", "list-item-2", "table-item-1", "tree-item-2", "tabs-item-2", "toolbar-item-1"} {
		found := false
		for i := 0; i < len(nodes); i++ {
			if nodes[i].ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("semantic tree missing %q: %#v", id, nodes)
		}
	}
}

func TestAdvancedControlsApplyCompleteThemeAndColumnLayout(t *testing.T) {
	var form Form
	form.Initialize(480, 240)
	table := NewListView()
	table.SetBounds(graphics.R(10, 10, 320, 120))
	table.AddColumn("Long property")
	table.AddColumn("Value")
	table.SetColumnWidth(0, 210)
	check := NewCheckBox()
	check.SetBounds(graphics.R(10, 150, 180, 30))
	check.SetChecked(true)
	form.Add(&table.Control)
	form.Add(&check.Control)

	dark := DarkTheme()
	form.ApplyTheme(dark)
	if form.Background() != dark.Window || table.Background() != dark.Field || table.Foreground() != dark.Text || check.Foreground() != dark.Text {
		t.Fatalf("dark theme was not applied consistently: form=%#v table=%#v/%#v check=%#v", form.Background(), table.Background(), table.Foreground(), check.Foreground())
	}
	if table.columnWidth(0, table.Bounds().Width()) != 210 || table.columnWidth(1, table.Bounds().Width()) != 110 {
		t.Fatalf("column widths = %v/%v", table.columnWidth(0, table.Bounds().Width()), table.columnWidth(1, table.Bounds().Width()))
	}

	surface := graphics.NewSurface(480, 240)
	if !form.Paint(surface) {
		t.Fatal("themed controls did not paint")
	}
	boxCenter := surfacePixel(surface, 19, 165)
	if boxCenter == dark.Field {
		t.Fatalf("checked checkbox center still looks unchecked: %#v", boxCenter)
	}
}

func TestComboBoxDropDownOwnsOverlappingInputAndDismisses(t *testing.T) {
	font := graphics.NewBuiltinFont(1)
	var form Form
	form.Initialize(420, 240)

	combo := NewComboBox()
	combo.SetBounds(graphics.R(10, 10, 180, 32))
	combo.SetFont(font)
	combo.AddItem("Debug")
	combo.AddItem("Release")
	combo.AddItem("Profile")
	combo.SetSelectedIndex(0)
	form.Add(&combo.Control)

	coveredClicks := 0
	covered := NewButton()
	covered.SetBounds(graphics.R(10, 42, 180, 90))
	covered.Click = func() { coveredClicks++ }
	form.Add(&covered.Control)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 30, Y: 20, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 30, Y: 20, Button: 1})
	if !combo.DroppedDown() || !combo.Control.popup {
		t.Fatal("combo box did not open its drop-down")
	}

	// The second row overlaps a later sibling in normal z-order. The popup must
	// still own both painting order and pointer routing.
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 30, Y: 87, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 30, Y: 87, Button: 1})
	if combo.SelectedIndex() != 1 || combo.SelectedItem() != "Release" || combo.DroppedDown() {
		t.Fatalf("drop-down selection = %d, %q, open=%v", combo.SelectedIndex(), combo.SelectedItem(), combo.DroppedDown())
	}
	if coveredClicks != 0 {
		t.Fatalf("covered sibling received %d popup clicks", coveredClicks)
	}

	combo.SetDroppedDown(true)
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 300, Y: 180, Button: 1})
	if combo.DroppedDown() {
		t.Fatal("outside pointer did not dismiss combo drop-down")
	}

	combo.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeySpace})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEscape})
	if combo.DroppedDown() {
		t.Fatal("escape did not dismiss keyboard-opened drop-down")
	}
}

func TestTreeViewExpansionPreservesHierarchyAndKeyboardNavigation(t *testing.T) {
	var form Form
	form.Initialize(360, 240)
	tree := NewTreeView()
	tree.SetBounds(graphics.R(10, 10, 260, 180))
	tree.AddNode("Project", 0)
	tree.AddNode("cmd", 1)
	tree.AddNode("main.go", 2)
	tree.AddNode("Documentation", 0)
	tree.AddNode("README.md", 1)
	form.Add(&tree.Control)

	tree.SetSelectedIndex(2)
	tree.SetExpanded(0, false)
	if tree.Expanded(0) || tree.SelectedIndex() != 0 || tree.visibleNodeCount() != 3 {
		t.Fatalf("collapsed tree: expanded=%v selected=%d visible=%d", tree.Expanded(0), tree.SelectedIndex(), tree.visibleNodeCount())
	}
	if !tree.nodeVisible(4) {
		t.Fatal("collapsing the first root hid a later root's child")
	}

	tree.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyRight})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyRight})
	if !tree.Expanded(0) || tree.SelectedIndex() != 1 {
		t.Fatalf("right-arrow expansion/navigation: expanded=%v selected=%d", tree.Expanded(0), tree.SelectedIndex())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyLeft})
	if tree.Expanded(1) || tree.SelectedIndex() != 1 {
		t.Fatalf("left arrow did not collapse selected branch: expanded=%v selected=%d", tree.Expanded(1), tree.SelectedIndex())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyLeft})
	if tree.SelectedIndex() != 0 {
		t.Fatalf("left arrow selected %d, want parent 0", tree.SelectedIndex())
	}

	tree.SetExpanded(1, true)
	nodes := tree.accessibilityChildren()
	if nodes[2].ParentID != widgetItemID(&tree.Control, 1) || nodes[2].Hidden {
		t.Fatalf("grandchild accessibility hierarchy = %#v", nodes[2])
	}
}

func TestAdvancedControlsImplementStandardKeyboardNavigation(t *testing.T) {
	var form Form
	form.Initialize(640, 400)

	list := NewListBox()
	list.SetBounds(graphics.R(0, 0, 140, 80))
	for _, item := range []string{"One", "Two", "Three", "Four", "Five"} {
		list.AddItem(item)
	}
	table := NewListView()
	table.SetBounds(graphics.R(150, 0, 200, 100))
	table.AddColumn("Name")
	for _, item := range []string{"One", "Two", "Three"} {
		table.AddRow([]string{item})
	}
	tabs := NewTabControl()
	tabs.SetBounds(graphics.R(0, 110, 300, 32))
	tabs.AddTab("One")
	tabs.AddTab("Two")
	tabs.AddTab("Three")
	number := NewNumericUpDown()
	number.SetBounds(graphics.R(0, 150, 100, 30))
	number.SetRange(10, 20)
	slider := NewSlider()
	slider.SetBounds(graphics.R(0, 190, 200, 30))
	slider.SetRange(10, 30)
	slider.SetValue(20)
	for _, control := range []*Control{&list.Control, &table.Control, &tabs.Control, &number.Control, &slider.Control} {
		form.Add(control)
	}

	list.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnd})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyPageUp})
	if list.SelectedIndex() != 1 {
		t.Fatalf("list PageUp selected %d, want 1", list.SelectedIndex())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "t"})
	if list.SelectedIndex() != 2 {
		t.Fatalf("list type-ahead selected %d, want 2", list.SelectedIndex())
	}
	table.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnd})
	if table.SelectedIndex() != 2 {
		t.Fatalf("list view End selected %d, want 2", table.SelectedIndex())
	}
	tabs.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyLeft})
	if tabs.SelectedIndex() != 2 {
		t.Fatalf("tab Left selected %d, want wrapped index 2", tabs.SelectedIndex())
	}
	number.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnd})
	if number.Value() != 20 {
		t.Fatalf("numeric End value %d, want 20", number.Value())
	}
	slider.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyRight})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyHome})
	if slider.Value() != 10 {
		t.Fatalf("slider Home value %d, want 10", slider.Value())
	}
}
