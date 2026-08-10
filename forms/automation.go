package forms

// AutomationDriver operates the same semantic tree exported to assistive
// technology. Tests therefore exercise named controls and real event handlers
// instead of maintaining a parallel coordinate-based automation model.
type AutomationDriver struct {
	form *Form
}

func NewAutomationDriver(form *Form) *AutomationDriver {
	return &AutomationDriver{form: form}
}

func (d *AutomationDriver) Nodes() []AccessibilityNode {
	if d == nil || d.form == nil {
		return nil
	}
	return d.form.AccessibilitySnapshot()
}

func (d *AutomationDriver) Node(id string) (AccessibilityNode, bool) {
	if d == nil || d.form == nil {
		return AccessibilityNode{}, false
	}
	return d.form.accessibilityNode(id)
}

func (d *AutomationDriver) Find(role AccessibilityRole, name string) []AccessibilityNode {
	nodes := d.Nodes()
	found := make([]AccessibilityNode, 0)
	for i := 0; i < len(nodes); i++ {
		if nodes[i].Role == role && (name == "" || nodes[i].Name == name) {
			found = append(found, nodes[i])
		}
	}
	return found
}

func (d *AutomationDriver) Invoke(id string) bool {
	return d != nil && d.form != nil && d.form.performAccessibilityAction(id, AccessibilityActionInvoke, "")
}

func (d *AutomationDriver) Focus(id string) bool {
	return d != nil && d.form != nil && d.form.performAccessibilityAction(id, AccessibilityActionFocus, "")
}

func (d *AutomationDriver) SetValue(id, value string) bool {
	return d != nil && d.form != nil && d.form.performAccessibilityAction(id, AccessibilityActionSetValue, value)
}

func (d *AutomationDriver) SetSelection(id string, start, end int) bool {
	value := accessibilityDecimal(start) + ":" + accessibilityDecimal(end)
	return d != nil && d.form != nil && d.form.performAccessibilityAction(id, AccessibilityActionSetSelection, value)
}
