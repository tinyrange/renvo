package forms

import "renvo.dev/std/graphics"

// AccessibilityRole describes the meaning of a retained control independently
// of how it is painted. Platform adapters translate these roles to their native
// accessibility representation.
type AccessibilityRole int

const (
	AccessibilityRoleGroup AccessibilityRole = iota
	AccessibilityRoleLabel
	AccessibilityRoleButton
	AccessibilityRoleTextBox
	AccessibilityRoleCheckBox
	AccessibilityRoleRadioButton
	AccessibilityRoleImage
	AccessibilityRoleMenuBar
	AccessibilityRoleMenu
	AccessibilityRoleMenuItem
	AccessibilityRoleSeparator
	AccessibilityRoleTree
	AccessibilityRoleTreeItem
	AccessibilityRoleList
	AccessibilityRoleListItem
	AccessibilityRoleStatus
)

// AccessibilityAction is an operation exposed by a semantic node.
type AccessibilityAction int

const (
	AccessibilityActionInvoke AccessibilityAction = iota + 1
	AccessibilityActionFocus
	AccessibilityActionSetValue
	AccessibilityActionSetSelection
)

const (
	AccessibilitySupportsInvoke = 1 << iota
	AccessibilitySupportsFocus
	AccessibilitySupportsSetValue
	AccessibilitySupportsSetSelection
)

// AccessibilityNode is the platform-neutral, immutable view of one control.
// Bounds use the same client-relative logical pixels as painting and input.
type AccessibilityNode struct {
	ID             string
	ParentID       string
	Role           AccessibilityRole
	Name           string
	Description    string
	Value          string
	Bounds         graphics.Rect
	Actions        int
	Hidden         bool
	Disabled       bool
	Focused        bool
	Checkable      bool
	Checked        bool
	Selectable     bool
	Selected       bool
	Multiline      bool
	SelectionStart int
	SelectionEnd   int
}

// AccessibilityUpdate contains only semantic nodes changed since the previous
// update. Reset asks an adapter to discard its old tree before applying Nodes.
type AccessibilityUpdate struct {
	Revision        int
	Reset           bool
	Nodes           []AccessibilityNode
	Removed         []string
	ReplaceChildren []string
	Selections      []AccessibilitySelectionUpdate
}

type AccessibilitySelectionUpdate struct {
	ID    string
	Start int
	End   int
}

func (c *Control) AccessibilityID() string { return c.accessibilityID }
func (c *Control) AccessibilityRole() AccessibilityRole {
	if c == nil {
		return AccessibilityRoleGroup
	}
	return c.accessibilityRole
}
func (c *Control) AccessibilityName() string {
	if c == nil {
		return ""
	}
	return c.accessibilityName
}
func (c *Control) AccessibilityDescription() string {
	if c == nil {
		return ""
	}
	return c.accessibilityDescription
}

func (c *Control) SetAccessibilityID(id string) {
	if c == nil || c.accessibilityID == id {
		return
	}
	if c.form != nil && c.accessibilityID != "" {
		c.form.accessibilityRemoved = append(c.form.accessibilityRemoved, c.accessibilityID)
	}
	c.accessibilityID = id
	if c.form != nil {
		c.accessibilityID = c.form.uniqueAccessibilityID(c, id)
		c.form.markAccessibilityChanged(c)
		c.form.markAccessibilityChildrenChanged(c)
	}
}

func (c *Control) SetAccessibilityRole(role AccessibilityRole) {
	if c == nil || c.accessibilityRole == role {
		return
	}
	c.accessibilityRole = role
	c.AccessibilityChanged()
}

func (c *Control) SetAccessibilityName(name string) {
	if c == nil || c.accessibilityName == name {
		return
	}
	c.accessibilityName = name
	c.AccessibilityChanged()
}

func (c *Control) SetAccessibilityDescription(description string) {
	if c == nil || c.accessibilityDescription == description {
		return
	}
	c.accessibilityDescription = description
	c.AccessibilityChanged()
}

func (c *Control) SetAccessibilityMultiline(multiline bool) {
	if c == nil || c.accessibilityMultiline == multiline {
		return
	}
	c.accessibilityMultiline = multiline
	c.AccessibilityChanged()
}

// AccessibilityChanged marks custom semantic state as dirty without causing a
// repaint. Custom controls call it when a value, selection, or virtual state
// exposed by one of the accessibility callbacks changes.
func (c *Control) AccessibilityChanged() {
	if c != nil && c.form != nil {
		c.form.markAccessibilityChanged(c)
	}
}

// AccessibilityChildrenChanged invalidates only a custom control's virtual
// descendants. The control's stable browser object and large values remain
// untouched when a completion list or diagnostic set changes.
func (c *Control) AccessibilityChildrenChanged() {
	if c != nil && c.form != nil && c.AccessibilityChildren != nil {
		c.form.markAccessibilityChildrenChanged(c)
	}
}

// AccessibilityChildrenStateChanged patches existing virtual descendants by
// stable ID. Use it when membership is unchanged, such as moving a list
// selection, so platform objects keep their identity.
func (c *Control) AccessibilityChildrenStateChanged() {
	if c != nil && c.form != nil && c.AccessibilityChildren != nil {
		c.form.markAccessibilityChildrenStateChanged(c)
	}
}

// AccessibilitySelectionChanged is a narrow mutation for caret movement. It
// avoids retransmitting a large text value when only its selection changed.
func (c *Control) AccessibilitySelectionChanged() {
	if c != nil && c.form != nil {
		c.form.markAccessibilitySelectionChanged(c)
	}
}

func (c *Control) accessibilityNode() AccessibilityNode {
	if c == nil {
		return AccessibilityNode{}
	}
	name := c.accessibilityName
	if name == "" && c.accessibilityRole != AccessibilityRoleTextBox {
		name = c.text
	}
	value := ""
	if c.AccessibilityValue != nil {
		value = c.AccessibilityValue()
	} else if c.accessibilityRole == AccessibilityRoleTextBox {
		value = c.text
	}
	checkable, checked := c.AccessibilityCheckable, false
	if c.AccessibilityChecked != nil {
		checked = c.AccessibilityChecked()
	}
	selectable, selected := c.AccessibilitySelectable, false
	if c.AccessibilitySelected != nil {
		selected = c.AccessibilitySelected()
	}
	actions := 0
	if c.AccessibilityInvoke != nil || c.Click != nil {
		actions |= AccessibilitySupportsInvoke
	}
	if c.tabStop {
		actions |= AccessibilitySupportsFocus
	}
	if c.AccessibilitySetValue != nil || c.accessibilityRole == AccessibilityRoleTextBox {
		actions |= AccessibilitySupportsSetValue
	}
	if c.AccessibilitySetSelection != nil {
		actions |= AccessibilitySupportsSetSelection
	}
	selectionStart, selectionEnd := -1, -1
	if c.AccessibilitySelectionStart != nil {
		selectionStart = c.AccessibilitySelectionStart()
	}
	if c.AccessibilitySelectionEnd != nil {
		selectionEnd = c.AccessibilitySelectionEnd()
	}
	return AccessibilityNode{
		ID:             c.accessibilityID,
		Role:           c.accessibilityRole,
		Name:           name,
		Description:    c.accessibilityDescription,
		Value:          value,
		Bounds:         c.bounds,
		Actions:        actions,
		Hidden:         !c.visible,
		Disabled:       !c.enabled,
		Focused:        c.Focused(),
		Checkable:      checkable,
		Checked:        checked,
		Selectable:     selectable,
		Selected:       selected,
		Multiline:      c.accessibilityMultiline,
		SelectionStart: selectionStart,
		SelectionEnd:   selectionEnd,
	}
}

func (c *Control) accessibilityNodes() []AccessibilityNode {
	base := c.accessibilityNode()
	nodes := []AccessibilityNode{base}
	return append(nodes, c.accessibilityChildNodes(base)...)
}

func (c *Control) accessibilityChildNodes(base AccessibilityNode) []AccessibilityNode {
	if c == nil || c.AccessibilityChildren == nil {
		return nil
	}
	children := c.AccessibilityChildren()
	nodes := make([]AccessibilityNode, 0, len(children))
	for i := 0; i < len(children); i++ {
		child := children[i]
		if child.ID == "" {
			child.ID = base.ID + "-item-" + accessibilityDecimal(i+1)
		}
		if child.ParentID == "" {
			child.ParentID = base.ID
		}
		child.Hidden = child.Hidden || base.Hidden
		child.Disabled = child.Disabled || base.Disabled
		nodes = append(nodes, child)
	}
	return nodes
}

func (c *Control) performAccessibilityAction(action AccessibilityAction, value string) bool {
	if c == nil || !c.visible || !c.enabled {
		return false
	}
	if action == AccessibilityActionFocus {
		if !c.tabStop {
			return false
		}
		c.Focus()
		return c.Focused()
	}
	if action == AccessibilityActionInvoke {
		if c.AccessibilityInvoke != nil {
			c.AccessibilityInvoke()
		} else if c.Click != nil {
			c.Click()
		} else {
			return false
		}
		c.AccessibilityChanged()
		return true
	}
	if action == AccessibilityActionSetValue {
		if c.AccessibilitySetValue != nil {
			c.AccessibilitySetValue(value)
		} else if c.accessibilityRole == AccessibilityRoleTextBox {
			c.SetText(value)
		} else {
			return false
		}
		c.AccessibilityChanged()
		return true
	}
	if action == AccessibilityActionSetSelection {
		if c.AccessibilitySetSelection == nil {
			return false
		}
		start, end, ok := accessibilitySelectionValue(value)
		if !ok {
			return false
		}
		c.AccessibilitySetSelection(start, end)
		c.AccessibilitySelectionChanged()
		return true
	}
	return false
}

// AccessibilitySnapshot returns the current semantic tree without consuming
// incremental changes. Controls are returned in retained z-order.
func (f *Form) AccessibilitySnapshot() []AccessibilityNode {
	if f == nil {
		return nil
	}
	nodes := make([]AccessibilityNode, 0, len(f.controls))
	for i := 0; i < len(f.controls); i++ {
		nodes = append(nodes, f.controls[i].accessibilityNodes()...)
	}
	return nodes
}

// TakeAccessibilityUpdate consumes the accumulated semantic mutations.
func (f *Form) TakeAccessibilityUpdate() (AccessibilityUpdate, bool) {
	if f == nil || !f.accessibilityReset && len(f.accessibilityDirty) == 0 && len(f.accessibilityChildrenDirty) == 0 && len(f.accessibilityChildrenPatchDirty) == 0 && len(f.accessibilitySelectionDirty) == 0 && len(f.accessibilityRemoved) == 0 {
		return AccessibilityUpdate{}, false
	}
	f.accessibilityRevision++
	update := AccessibilityUpdate{Revision: f.accessibilityRevision, Reset: f.accessibilityReset}
	if !f.accessibilityReset {
		for i := 0; i < len(f.accessibilitySelectionDirty); i++ {
			control := f.accessibilitySelectionDirty[i]
			if control.form != f {
				continue
			}
			if f.accessibilityControlDirty(control) {
				continue
			}
			start, end := -1, -1
			if control.AccessibilitySelectionStart != nil {
				start = control.AccessibilitySelectionStart()
			}
			if control.AccessibilitySelectionEnd != nil {
				end = control.AccessibilitySelectionEnd()
			}
			update.Selections = append(update.Selections, AccessibilitySelectionUpdate{ID: control.accessibilityID, Start: start, End: end})
		}
	}
	if f.accessibilityReset {
		update.Nodes = f.AccessibilitySnapshot()
	} else {
		for i := 0; i < len(f.accessibilityDirty); i++ {
			control := f.accessibilityDirty[i]
			if control.form == f {
				update.Nodes = append(update.Nodes, control.accessibilityNode())
			}
		}
		for i := 0; i < len(f.accessibilityChildrenDirty); i++ {
			control := f.accessibilityChildrenDirty[i]
			if control.form == f && control.AccessibilityChildren != nil {
				update.Nodes = append(update.Nodes, control.accessibilityChildNodes(control.accessibilityNode())...)
				update.ReplaceChildren = append(update.ReplaceChildren, control.accessibilityID)
			}
		}
		for i := 0; i < len(f.accessibilityChildrenPatchDirty); i++ {
			control := f.accessibilityChildrenPatchDirty[i]
			if control.form == f && control.AccessibilityChildren != nil && !f.accessibilityChildrenControlDirty(control) {
				update.Nodes = append(update.Nodes, control.accessibilityChildNodes(control.accessibilityNode())...)
			}
		}
	}
	update.Removed = append(update.Removed, f.accessibilityRemoved...)
	f.accessibilityReset = false
	f.accessibilityDirty = nil
	f.accessibilityChildrenDirty = nil
	f.accessibilityChildrenPatchDirty = nil
	f.accessibilitySelectionDirty = nil
	f.accessibilityRemoved = nil
	return update, true
}

func (f *Form) accessibilityControlDirty(control *Control) bool {
	for i := 0; i < len(f.accessibilityDirty); i++ {
		if f.accessibilityDirty[i] == control {
			return true
		}
	}
	return false
}

func (f *Form) markAccessibilitySelectionChanged(control *Control) {
	if f == nil || control == nil || control.form != f || f.accessibilityReset || f.accessibilityControlDirty(control) {
		return
	}
	for i := 0; i < len(f.accessibilitySelectionDirty); i++ {
		if f.accessibilitySelectionDirty[i] == control {
			return
		}
	}
	f.accessibilitySelectionDirty = append(f.accessibilitySelectionDirty, control)
}

func (f *Form) markAccessibilityChildrenChanged(control *Control) {
	if f == nil || control == nil || control.form != f || control.AccessibilityChildren == nil || f.accessibilityReset {
		return
	}
	for i := 0; i < len(f.accessibilityChildrenDirty); i++ {
		if f.accessibilityChildrenDirty[i] == control {
			return
		}
	}
	f.accessibilityChildrenDirty = append(f.accessibilityChildrenDirty, control)
}

func (f *Form) accessibilityChildrenControlDirty(control *Control) bool {
	for i := 0; i < len(f.accessibilityChildrenDirty); i++ {
		if f.accessibilityChildrenDirty[i] == control {
			return true
		}
	}
	return false
}

func (f *Form) markAccessibilityChildrenStateChanged(control *Control) {
	if f == nil || control == nil || control.form != f || control.AccessibilityChildren == nil || f.accessibilityReset || f.accessibilityChildrenControlDirty(control) {
		return
	}
	for i := 0; i < len(f.accessibilityChildrenPatchDirty); i++ {
		if f.accessibilityChildrenPatchDirty[i] == control {
			return
		}
	}
	f.accessibilityChildrenPatchDirty = append(f.accessibilityChildrenPatchDirty, control)
}

func (f *Form) markAccessibilityChanged(control *Control) {
	if f == nil || control == nil || control.form != f || f.accessibilityReset {
		return
	}
	for i := 0; i < len(f.accessibilityDirty); i++ {
		if f.accessibilityDirty[i] == control {
			return
		}
	}
	f.accessibilityDirty = append(f.accessibilityDirty, control)
}

func (f *Form) uniqueAccessibilityID(control *Control, requested string) string {
	base := requested
	if base == "" {
		base = "control-" + accessibilityDecimal(f.accessibilityNextID)
		f.accessibilityNextID++
	}
	id := base
	suffix := 2
	for {
		used := false
		for i := 0; i < len(f.controls); i++ {
			if f.controls[i] != control && f.controls[i].accessibilityID == id {
				used = true
				break
			}
		}
		if !used {
			return id
		}
		id = base + "-" + accessibilityDecimal(suffix)
		suffix++
	}
}

func accessibilityDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

func (f *Form) accessibilityControl(id string) *Control {
	if f == nil || id == "" {
		return nil
	}
	for i := 0; i < len(f.controls); i++ {
		if f.controls[i].accessibilityID == id {
			return f.controls[i]
		}
	}
	return nil
}

func (f *Form) accessibilityNode(id string) (AccessibilityNode, bool) {
	if f == nil || id == "" {
		return AccessibilityNode{}, false
	}
	for i := 0; i < len(f.controls); i++ {
		nodes := f.controls[i].accessibilityNodes()
		for j := 0; j < len(nodes); j++ {
			if nodes[j].ID == id {
				return nodes[j], true
			}
		}
	}
	return AccessibilityNode{}, false
}

func (f *Form) performAccessibilityAction(id string, action AccessibilityAction, value string) bool {
	control := f.accessibilityControl(id)
	if control != nil {
		return control.performAccessibilityAction(action, value)
	}
	for i := 0; i < len(f.controls); i++ {
		control = f.controls[i]
		if control.visible && control.enabled && control.AccessibilityPerform != nil && control.AccessibilityPerform(id, action, value) {
			return true
		}
	}
	return false
}

func accessibilityActionPayload(payload string) (string, string) {
	for i := 0; i < len(payload); i++ {
		if payload[i] == 0 {
			return payload[:i], payload[i+1:]
		}
	}
	return payload, ""
}

func accessibilitySelectionValue(value string) (int, int, bool) {
	separator := -1
	for i := 0; i < len(value); i++ {
		if value[i] == ':' {
			separator = i
			break
		}
	}
	if separator <= 0 || separator == len(value)-1 {
		return 0, 0, false
	}
	start, ok := accessibilityParseDecimal(value[:separator])
	if !ok {
		return 0, 0, false
	}
	end, ok := accessibilityParseDecimal(value[separator+1:])
	if !ok || end < start {
		return 0, 0, false
	}
	return start, end, true
}

func accessibilityParseDecimal(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		result = result*10 + int(value[i]-'0')
	}
	return result, true
}

const (
	accessibilityStateHidden = 1 << iota
	accessibilityStateDisabled
	accessibilityStateFocused
	accessibilityStateCheckable
	accessibilityStateChecked
	accessibilityStateSelectable
	accessibilityStateSelected
	accessibilityStateMultiline
)

// encodeAccessibilityUpdate is intentionally a small binary protocol. It
// avoids a JSON dependency in embedded applications and lets browser adapters
// patch only controls whose retained properties changed.
func encodeAccessibilityUpdate(update AccessibilityUpdate) []byte {
	data := make([]byte, 0, 64+len(update.Nodes)*64)
	data = accessibilityAppend32(data, 1)
	data = accessibilityAppend32(data, update.Revision)
	flags := 0
	if update.Reset {
		flags = 1
	}
	data = accessibilityAppend32(data, flags)
	data = accessibilityAppend32(data, len(update.Removed))
	for i := 0; i < len(update.Removed); i++ {
		data = accessibilityAppendString(data, update.Removed[i])
	}
	data = accessibilityAppend32(data, len(update.ReplaceChildren))
	for i := 0; i < len(update.ReplaceChildren); i++ {
		data = accessibilityAppendString(data, update.ReplaceChildren[i])
	}
	data = accessibilityAppend32(data, len(update.Selections))
	for i := 0; i < len(update.Selections); i++ {
		data = accessibilityAppendString(data, update.Selections[i].ID)
		data = accessibilityAppend32(data, update.Selections[i].Start)
		data = accessibilityAppend32(data, update.Selections[i].End)
	}
	data = accessibilityAppend32(data, len(update.Nodes))
	for i := 0; i < len(update.Nodes); i++ {
		node := update.Nodes[i]
		data = accessibilityAppendString(data, node.ID)
		data = accessibilityAppendString(data, node.ParentID)
		data = accessibilityAppend32(data, int(node.Role))
		data = accessibilityAppendString(data, node.Name)
		data = accessibilityAppendString(data, node.Description)
		data = accessibilityAppendString(data, node.Value)
		data = accessibilityAppend32(data, int(node.Bounds.MinX))
		data = accessibilityAppend32(data, int(node.Bounds.MinY))
		data = accessibilityAppend32(data, int(node.Bounds.Width()))
		data = accessibilityAppend32(data, int(node.Bounds.Height()))
		data = accessibilityAppend32(data, node.Actions)
		state := 0
		if node.Hidden {
			state |= accessibilityStateHidden
		}
		if node.Disabled {
			state |= accessibilityStateDisabled
		}
		if node.Focused {
			state |= accessibilityStateFocused
		}
		if node.Checkable {
			state |= accessibilityStateCheckable
		}
		if node.Checked {
			state |= accessibilityStateChecked
		}
		if node.Selectable {
			state |= accessibilityStateSelectable
		}
		if node.Selected {
			state |= accessibilityStateSelected
		}
		if node.Multiline {
			state |= accessibilityStateMultiline
		}
		data = accessibilityAppend32(data, state)
		data = accessibilityAppend32(data, node.SelectionStart)
		data = accessibilityAppend32(data, node.SelectionEnd)
	}
	return data
}

func accessibilityAppend32(data []byte, value int) []byte {
	return append(data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func accessibilityAppendString(data []byte, value string) []byte {
	data = accessibilityAppend32(data, len(value))
	return append(data, []byte(value)...)
}
