//go:build renvo && darwin && arm64

package graphics

// renvo:linkstatic /System/Library/Frameworks/AppKit.framework/AppKit,NSAccessibilityPostNotification
func nsAccessibilityPostNotification(element, notification int) {}

const (
	darwinAccessibilityRoleGroup = iota
	darwinAccessibilityRoleLabel
	darwinAccessibilityRoleButton
	darwinAccessibilityRoleTextBox
	darwinAccessibilityRoleCheckBox
	darwinAccessibilityRoleRadioButton
	darwinAccessibilityRoleImage
	darwinAccessibilityRoleMenuBar
	darwinAccessibilityRoleMenu
	darwinAccessibilityRoleMenuItem
	darwinAccessibilityRoleSeparator
	darwinAccessibilityRoleTree
	darwinAccessibilityRoleTreeItem
	darwinAccessibilityRoleList
	darwinAccessibilityRoleListItem
	darwinAccessibilityRoleStatus
)

const (
	darwinAccessibilityStateHidden = 1 << iota
	darwinAccessibilityStateDisabled
	darwinAccessibilityStateFocused
	darwinAccessibilityStateCheckable
	darwinAccessibilityStateChecked
	darwinAccessibilityStateSelectable
	darwinAccessibilityStateSelected
	darwinAccessibilityStateMultiline
)

type darwinAccessibilityNode struct {
	id             string
	parent         string
	native         int
	role           int
	name           string
	description    string
	value          string
	x              int
	y              int
	width          int
	height         int
	actions        int
	state          int
	selectionStart int
	selectionEnd   int
}

type darwinAccessibilityState struct {
	window   *Window
	revision int
	nodes    []*darwinAccessibilityNode
}

type darwinAccessibilityReader struct {
	data []byte
	at   int
	ok   bool
}

var darwinAccessibilityStates []*darwinAccessibilityState

func darwinAccessibilityStateFor(w *Window, create bool) *darwinAccessibilityState {
	for i := 0; i < len(darwinAccessibilityStates); i++ {
		if darwinAccessibilityStates[i].window == w {
			return darwinAccessibilityStates[i]
		}
	}
	if !create {
		return nil
	}
	state := &darwinAccessibilityState{window: w}
	darwinAccessibilityStates = append(darwinAccessibilityStates, state)
	return state
}

func darwinAccessibilityPollDelay(w *Window) Scalar {
	// Native accessibility actions do not yet bridge back into the Forms event
	// queue. Until that bridge exists there is nothing to poll, and scheduling a
	// periodic Cocoa deadline needlessly wakes an otherwise idle application.
	return -1.0
}

func (r *darwinAccessibilityReader) unsigned() int {
	if !r.ok || r.at+4 > len(r.data) {
		r.ok = false
		return 0
	}
	value := int(r.data[r.at]) | int(r.data[r.at+1])<<8 | int(r.data[r.at+2])<<16 | int(r.data[r.at+3])<<24
	r.at += 4
	return value
}

func (r *darwinAccessibilityReader) signed() int {
	value := r.unsigned()
	if value >= 1<<31 {
		value -= 1 << 32
	}
	return value
}

func (r *darwinAccessibilityReader) text() string {
	length := r.unsigned()
	if !r.ok || length < 0 || r.at+length > len(r.data) {
		r.ok = false
		return ""
	}
	value := string(r.data[r.at : r.at+length])
	r.at += length
	return value
}

func darwinAccessibilityFind(state *darwinAccessibilityState, id string) *darwinAccessibilityNode {
	if state == nil || id == "" {
		return nil
	}
	for i := 0; i < len(state.nodes); i++ {
		if state.nodes[i].id == id {
			return state.nodes[i]
		}
	}
	return nil
}

func darwinAccessibilityRole(role, state int) string {
	if role == darwinAccessibilityRoleLabel || role == darwinAccessibilityRoleStatus {
		return "AXStaticText"
	}
	if role == darwinAccessibilityRoleButton {
		return "AXButton"
	}
	if role == darwinAccessibilityRoleTextBox {
		if state&darwinAccessibilityStateMultiline == 0 {
			return "AXTextField"
		}
		return "AXTextArea"
	}
	if role == darwinAccessibilityRoleCheckBox {
		return "AXCheckBox"
	}
	if role == darwinAccessibilityRoleRadioButton {
		return "AXRadioButton"
	}
	if role == darwinAccessibilityRoleImage {
		return "AXImage"
	}
	if role == darwinAccessibilityRoleMenuBar {
		return "AXMenuBar"
	}
	if role == darwinAccessibilityRoleMenu {
		return "AXMenu"
	}
	if role == darwinAccessibilityRoleMenuItem {
		return "AXMenuItem"
	}
	if role == darwinAccessibilityRoleSeparator {
		return "AXSplitter"
	}
	if role == darwinAccessibilityRoleTree {
		return "AXOutline"
	}
	if role == darwinAccessibilityRoleTreeItem || role == darwinAccessibilityRoleListItem {
		return "AXRow"
	}
	if role == darwinAccessibilityRoleList {
		return "AXList"
	}
	return "AXGroup"
}

func darwinAccessibilityUTF16Offset(value string, byteOffset int) int {
	if byteOffset < 0 {
		return 0
	}
	if byteOffset > len(value) {
		byteOffset = len(value)
	}
	units := 0
	for at := 0; at < byteOffset; {
		size := 1
		wide := false
		first := value[at]
		if first&0xe0 == 0xc0 {
			size = 2
		} else if first&0xf0 == 0xe0 {
			size = 3
		} else if first&0xf8 == 0xf0 {
			size = 4
			wide = true
		}
		if at+size > byteOffset {
			break
		}
		if wide {
			units += 2
		} else {
			units++
		}
		at += size
	}
	return units
}

func darwinAccessibilitySetSelection(node *darwinAccessibilityNode) {
	if node == nil || node.native == 0 || node.selectionStart < 0 || node.selectionEnd < node.selectionStart {
		return
	}
	start := darwinAccessibilityUTF16Offset(node.value, node.selectionStart)
	end := darwinAccessibilityUTF16Offset(node.value, node.selectionEnd)
	objcMsg2(node.native, selector("setAccessibilitySelectedTextRange:"), start, end-start)
	objcMsg1(node.native, selector("setAccessibilityNumberOfCharacters:"), darwinAccessibilityUTF16Offset(node.value, len(node.value)))
}

func darwinAccessibilityApplyNode(state *darwinAccessibilityState, next darwinAccessibilityNode) (bool, bool, bool) {
	node := darwinAccessibilityFind(state, next.id)
	created := node == nil
	if created {
		node = &darwinAccessibilityNode{id: next.id}
		node.native = objcMsg0(objcGetClass("NSAccessibilityElement"), selector("alloc"))
		node.native = objcMsg0(node.native, selector("init"))
		if node.native == 0 {
			return false, false, false
		}
		state.nodes = append(state.nodes, node)
	}
	oldParent := node.parent
	oldValue := node.value
	oldHidden := node.state&darwinAccessibilityStateHidden != 0
	oldFocused := node.state&darwinAccessibilityStateFocused != 0
	oldX, oldY, oldWidth, oldHeight := node.x, node.y, node.width, node.height
	native := node.native
	*node = next
	node.native = native

	objcMsg1(native, selector("setAccessibilityElement:"), 1)
	objcMsg1(native, selector("setAccessibilityIdentifier:"), cocoaString(node.id))
	objcMsg1(native, selector("setAccessibilityRole:"), cocoaString(darwinAccessibilityRole(node.role, node.state)))
	objcMsg1(native, selector("setAccessibilityLabel:"), cocoaString(node.name))
	objcMsg1(native, selector("setAccessibilityHelp:"), cocoaString(node.description))
	value := cocoaString(node.value)
	if node.state&darwinAccessibilityStateCheckable != 0 {
		checked := 0
		if node.state&darwinAccessibilityStateChecked != 0 {
			checked = 1
		}
		value = objcMsg1(objcGetClass("NSNumber"), selector("numberWithBool:"), checked)
	}
	objcMsg1(native, selector("setAccessibilityValue:"), value)
	objcMsg1(native, selector("setAccessibilityEnabled:"), boolInt(node.state&darwinAccessibilityStateDisabled == 0))
	objcMsg1(native, selector("setAccessibilityFocused:"), boolInt(node.state&darwinAccessibilityStateFocused != 0))
	objcMsg1(native, selector("setAccessibilitySelected:"), boolInt(node.state&darwinAccessibilityStateSelected != 0))
	objcMsg1(native, selector("setAccessibilityHidden:"), boolInt(node.state&darwinAccessibilityStateHidden != 0))
	darwinAccessibilitySetSelection(node)

	valueChanged := !created && oldValue != node.value
	focusChanged := !created && oldFocused != (node.state&darwinAccessibilityStateFocused != 0)
	geometryChanged := !created && (oldX != node.x || oldY != node.y || oldWidth != node.width || oldHeight != node.height)
	return created || oldParent != node.parent || oldHidden != (node.state&darwinAccessibilityStateHidden != 0), valueChanged, focusChanged || geometryChanged
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func darwinAccessibilityRemove(state *darwinAccessibilityState, id string) bool {
	if state == nil || id == "" {
		return false
	}
	removed := false
	for {
		child := ""
		for i := 0; i < len(state.nodes); i++ {
			if state.nodes[i].parent == id {
				child = state.nodes[i].id
				break
			}
		}
		if child == "" {
			break
		}
		darwinAccessibilityRemove(state, child)
	}
	for i := 0; i < len(state.nodes); i++ {
		if state.nodes[i].id == id {
			node := state.nodes[i]
			state.nodes = append(state.nodes[:i], state.nodes[i+1:]...)
			if node.native != 0 {
				nsAccessibilityPostNotification(node.native, cocoaString("AXUIElementDestroyed"))
				objcMsg0(node.native, selector("release"))
			}
			removed = true
			break
		}
	}
	return removed
}

func darwinAccessibilityClear(state *darwinAccessibilityState) {
	if state == nil {
		return
	}
	if state.window != nil && state.window.view != 0 {
		objcMsg1(state.window.view, selector("setAccessibilityChildren:"), objcMsg0(objcGetClass("NSArray"), selector("array")))
	}
	for i := 0; i < len(state.nodes); i++ {
		if state.nodes[i].native != 0 {
			objcMsg0(state.nodes[i].native, selector("release"))
		}
	}
	state.nodes = nil
}

func darwinAccessibilityChildren(state *darwinAccessibilityState, parent string) int {
	array := objcMsg1(objcGetClass("NSMutableArray"), selector("arrayWithCapacity:"), len(state.nodes))
	for i := 0; i < len(state.nodes); i++ {
		node := state.nodes[i]
		if node.parent == parent && node.native != 0 && darwinAccessibilityVisible(state, node) {
			objcMsg1(array, selector("addObject:"), node.native)
		}
	}
	return array
}

func darwinAccessibilityVisible(state *darwinAccessibilityState, node *darwinAccessibilityNode) bool {
	for node != nil {
		if node.state&darwinAccessibilityStateHidden != 0 {
			return false
		}
		node = darwinAccessibilityFind(state, node.parent)
	}
	return true
}

func darwinAccessibilitySyncTopology(state *darwinAccessibilityState) {
	if state == nil || state.window == nil || state.window.view == 0 {
		return
	}
	view := state.window.view
	objcMsg1(view, selector("setAccessibilityElement:"), 1)
	objcMsg1(view, selector("setAccessibilityRole:"), cocoaString("AXGroup"))
	objcMsg1(view, selector("setAccessibilityChildren:"), darwinAccessibilityChildren(state, ""))
	for i := 0; i < len(state.nodes); i++ {
		node := state.nodes[i]
		parentNative := view
		parentX, parentY, parentHeight := 0, 0, state.window.height
		if parent := darwinAccessibilityFind(state, node.parent); parent != nil {
			parentNative = parent.native
			parentX, parentY, parentHeight = parent.x, parent.y, parent.height
		}
		objcMsg1(node.native, selector("setAccessibilityParent:"), parentNative)
		objcMsg1(node.native, selector("setAccessibilityWindow:"), state.window.native)
		objcMsg1(node.native, selector("setAccessibilityTopLevelUIElement:"), state.window.native)
		objcMsg1(node.native, selector("setAccessibilityChildren:"), darwinAccessibilityChildren(state, node.id))
		x := node.x - parentX
		y := parentHeight - (node.y - parentY) - node.height
		objcMsgRect(node.native, selector("setAccessibilityFrameInParentSpace:"), x, y, node.width, node.height, 0, 0)
	}
	nsAccessibilityPostNotification(view, cocoaString("AXLayoutChanged"))
}

// PresentAccessibility applies the same compact semantic-tree patches used by
// the browser adapter to AppKit's native NSAccessibilityElement graph.
func (w *Window) PresentAccessibility(data []byte) bool {
	if w == nil || w.closed || w.view == 0 || len(data) < 20 {
		return false
	}
	reader := darwinAccessibilityReader{data: data, ok: true}
	if reader.unsigned() != 1 {
		return false
	}
	revision := reader.unsigned()
	flags := reader.unsigned()
	removedCount := reader.unsigned()
	removed := make([]string, removedCount)
	for i := 0; i < removedCount; i++ {
		removed[i] = reader.text()
	}
	replacedCount := reader.unsigned()
	replaced := make([]string, replacedCount)
	for i := 0; i < replacedCount; i++ {
		replaced[i] = reader.text()
	}
	selectionCount := reader.unsigned()
	selectionIDs := make([]string, selectionCount)
	selectionStarts := make([]int, selectionCount)
	selectionEnds := make([]int, selectionCount)
	for i := 0; i < selectionCount; i++ {
		selectionIDs[i] = reader.text()
		selectionStarts[i] = reader.signed()
		selectionEnds[i] = reader.signed()
	}
	nodeCount := reader.unsigned()
	nodes := make([]darwinAccessibilityNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = darwinAccessibilityNode{
			id:             reader.text(),
			parent:         reader.text(),
			role:           reader.unsigned(),
			name:           reader.text(),
			description:    reader.text(),
			value:          reader.text(),
			x:              reader.signed(),
			y:              reader.signed(),
			width:          reader.signed(),
			height:         reader.signed(),
			actions:        reader.unsigned(),
			state:          reader.unsigned(),
			selectionStart: reader.signed(),
			selectionEnd:   reader.signed(),
		}
		if nodes[i].id == "" {
			reader.ok = false
		}
	}
	if !reader.ok || reader.at != len(data) {
		return false
	}
	state := darwinAccessibilityStateFor(w, true)
	if state == nil || revision <= state.revision {
		return state != nil
	}
	pool := objcMsg0(objcGetClass("NSAutoreleasePool"), selector("alloc"))
	pool = objcMsg0(pool, selector("init"))
	topologyChanged := flags&1 != 0
	if topologyChanged {
		darwinAccessibilityClear(state)
	}
	for i := 0; i < len(removed); i++ {
		topologyChanged = darwinAccessibilityRemove(state, removed[i]) || topologyChanged
	}
	for i := 0; i < len(replaced); i++ {
		for {
			child := ""
			for j := 0; j < len(state.nodes); j++ {
				if state.nodes[j].parent == replaced[i] {
					child = state.nodes[j].id
					break
				}
			}
			if child == "" {
				break
			}
			darwinAccessibilityRemove(state, child)
			topologyChanged = true
		}
	}
	for i := 0; i < len(selectionIDs); i++ {
		if node := darwinAccessibilityFind(state, selectionIDs[i]); node != nil {
			node.selectionStart, node.selectionEnd = selectionStarts[i], selectionEnds[i]
			darwinAccessibilitySetSelection(node)
			nsAccessibilityPostNotification(node.native, cocoaString("AXSelectedTextChanged"))
		}
	}
	for i := 0; i < len(nodes); i++ {
		changedTopology, valueChanged, presentationChanged := darwinAccessibilityApplyNode(state, nodes[i])
		topologyChanged = topologyChanged || changedTopology
		node := darwinAccessibilityFind(state, nodes[i].id)
		if node != nil && !topologyChanged {
			if valueChanged {
				nsAccessibilityPostNotification(node.native, cocoaString("AXValueChanged"))
			}
			if presentationChanged {
				nsAccessibilityPostNotification(node.native, cocoaString("AXLayoutChanged"))
			}
		}
	}
	if topologyChanged {
		darwinAccessibilitySyncTopology(state)
	} else {
		// Bounds are parent-relative, so a geometry-only patch still needs the
		// frame setter even though object ownership did not change.
		for i := 0; i < len(nodes); i++ {
			node := darwinAccessibilityFind(state, nodes[i].id)
			if node == nil {
				continue
			}
			parentX, parentY, parentHeight := 0, 0, w.height
			if parent := darwinAccessibilityFind(state, node.parent); parent != nil {
				parentX, parentY, parentHeight = parent.x, parent.y, parent.height
			}
			objcMsgRect(node.native, selector("setAccessibilityFrameInParentSpace:"), node.x-parentX, parentHeight-(node.y-parentY)-node.height, node.width, node.height, 0, 0)
		}
	}
	state.revision = revision
	if pool != 0 {
		objcMsg0(pool, selector("drain"))
	}
	return true
}

func darwinAccessibilityForget(w *Window) {
	state := darwinAccessibilityStateFor(w, false)
	if state == nil {
		return
	}
	darwinAccessibilityClear(state)
	for i := 0; i < len(darwinAccessibilityStates); i++ {
		if darwinAccessibilityStates[i] == state {
			darwinAccessibilityStates = append(darwinAccessibilityStates[:i], darwinAccessibilityStates[i+1:]...)
			return
		}
	}
}
