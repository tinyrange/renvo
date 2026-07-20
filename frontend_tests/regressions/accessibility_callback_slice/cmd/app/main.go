package main

import "renvo.dev/forms"

type customControl struct {
	forms.Control
}

func (c *customControl) children() []forms.AccessibilityNode {
	base := c.AccessibilityID()
	return []forms.AccessibilityNode{
		{ID: base + "-one", Name: "One"},
		{ID: base + "-two", Name: "Two"},
		{ID: base + "-three", Name: "Three"},
	}
}

func main() {
	var form forms.Form
	form.Initialize(100, 100)
	control := &customControl{}
	control.Control = *forms.NewControl()
	control.SetAccessibilityID("probe")
	control.AccessibilityChildren = control.children
	form.Add(&control.Control)

	direct := control.AccessibilityChildren()
	nodes := form.AccessibilitySnapshot()
	if len(direct) == 3 && direct[0].ID == "probe-one" && len(nodes) == 4 && nodes[1].ID == "probe-one" && nodes[1].ParentID == "probe" && nodes[3].ID == "probe-three" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
