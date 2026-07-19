package ide

import renvoos "renvo.dev/std/os"

// ExplorerNode is one file or directory in a project tree. Directories are
// loaded lazily so opening a project does not walk build outputs or module
// caches that the user may never inspect.
type ExplorerNode struct {
	Name      string
	Path      string
	Directory bool
	Expanded  bool
	Loaded    bool
	Error     string
	Parent    *ExplorerNode
	Children  []*ExplorerNode
}

// ExplorerRow is the visible projection of an explorer node.
type ExplorerRow struct {
	Node  *ExplorerNode
	Depth int
}

// Explorer owns expansion and selection state for a project directory.
type Explorer struct {
	Root       *ExplorerNode
	ShowHidden bool
	rows       []ExplorerRow
	selected   int
}

// OpenExplorer opens root and loads its immediate children. Directory
// failures are retained on their nodes so the view can display useful context.
func OpenExplorer(root string) *Explorer {
	root = trimTrailingSeparators(root)
	name := pathBase(root)
	if name == "" {
		name = root
	}
	node := &ExplorerNode{Name: name, Path: root, Directory: true, Expanded: true}
	explorer := &Explorer{Root: node, selected: 0}
	explorer.load(node, false)
	explorer.rebuildRows("")
	return explorer
}

// Rows returns the current visible tree projection. The returned slice is a
// copy so views cannot corrupt explorer navigation state.
func (e *Explorer) Rows() []ExplorerRow {
	if e == nil {
		return nil
	}
	rows := make([]ExplorerRow, len(e.rows))
	copy(rows, e.rows)
	return rows
}

func (e *Explorer) SelectedIndex() int {
	if e == nil || len(e.rows) == 0 {
		return -1
	}
	return e.selected
}

func (e *Explorer) Selected() *ExplorerNode {
	if e == nil || e.selected < 0 || e.selected >= len(e.rows) {
		return nil
	}
	return e.rows[e.selected].Node
}

func (e *Explorer) Select(index int) {
	if e == nil || len(e.rows) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(e.rows) {
		index = len(e.rows) - 1
	}
	e.selected = index
}

func (e *Explorer) Move(delta int) {
	if e == nil {
		return
	}
	e.Select(e.selected + delta)
}

func (e *Explorer) First() { e.Select(0) }

func (e *Explorer) Last() {
	if e != nil {
		e.Select(len(e.rows) - 1)
	}
}

// ExpandOrChild implements conventional tree Right-arrow behavior.
func (e *Explorer) ExpandOrChild() {
	node := e.Selected()
	if node == nil || !node.Directory {
		return
	}
	if !node.Expanded {
		e.setExpanded(node, true)
		return
	}
	if len(node.Children) != 0 {
		for i := 0; i < len(e.rows); i++ {
			if e.rows[i].Node == node.Children[0] {
				e.selected = i
				return
			}
		}
	}
}

// CollapseOrParent implements conventional tree Left-arrow behavior.
func (e *Explorer) CollapseOrParent() {
	node := e.Selected()
	if node == nil {
		return
	}
	if node.Directory && node.Expanded {
		e.setExpanded(node, false)
		return
	}
	if node.Parent == nil {
		return
	}
	for i := 0; i < len(e.rows); i++ {
		if e.rows[i].Node == node.Parent {
			e.selected = i
			return
		}
	}
}

func (e *Explorer) ToggleSelected() {
	node := e.Selected()
	if node != nil && node.Directory {
		e.setExpanded(node, !node.Expanded)
	}
}

// ActivateSelected toggles a directory or returns the selected file path.
func (e *Explorer) ActivateSelected() (string, bool) {
	node := e.Selected()
	if node == nil {
		return "", false
	}
	if node.Directory {
		e.setExpanded(node, !node.Expanded)
		return "", false
	}
	return node.Path, true
}

func (e *Explorer) SetShowHidden(show bool) {
	if e == nil || e.ShowHidden == show {
		return
	}
	selected := ""
	if node := e.Selected(); node != nil {
		selected = node.Path
	}
	e.ShowHidden = show
	e.refreshNode(e.Root)
	e.rebuildRows(selected)
}

// Refresh reloads every directory which has already been visited. Reused
// nodes retain expansion state, and selection is restored by stable path.
func (e *Explorer) Refresh() {
	if e == nil || e.Root == nil {
		return
	}
	selected := ""
	if node := e.Selected(); node != nil {
		selected = node.Path
	}
	e.refreshNode(e.Root)
	e.rebuildRows(selected)
}

func (e *Explorer) setExpanded(node *ExplorerNode, expanded bool) {
	selected := ""
	if current := e.Selected(); current != nil {
		selected = current.Path
	}
	if expanded && !node.Loaded {
		e.load(node, false)
	}
	node.Expanded = expanded
	e.rebuildRows(selected)
}

func (e *Explorer) refreshNode(node *ExplorerNode) {
	if node == nil || !node.Directory || !node.Loaded {
		return
	}
	e.load(node, true)
	for i := 0; i < len(node.Children); i++ {
		e.refreshNode(node.Children[i])
	}
}

func (e *Explorer) load(node *ExplorerNode, refresh bool) {
	entries, err := renvoos.ReadDir(node.Path)
	if err != nil {
		node.Error = err.Error()
		node.Loaded = true
		return
	}
	old := node.Children
	children := make([]*ExplorerNode, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		name := entries[i].Name()
		if !e.ShowHidden && len(name) > 0 && name[0] == '.' {
			continue
		}
		path := joinPath(node.Path, name)
		var child *ExplorerNode
		if refresh {
			for j := 0; j < len(old); j++ {
				if old[j].Path == path {
					child = old[j]
					break
				}
			}
		}
		if child == nil {
			child = &ExplorerNode{Name: name, Path: path, Parent: node}
		}
		child.Name = name
		child.Parent = node
		child.Directory = entries[i].IsDir()
		if !child.Directory {
			child.Expanded = false
			child.Loaded = false
			child.Children = nil
			child.Error = ""
		}
		children = append(children, child)
	}
	sortExplorerNodes(children)
	node.Children = children
	node.Error = ""
	node.Loaded = true
}

func (e *Explorer) rebuildRows(selectedPath string) {
	e.rows = e.rows[:0]
	if e.Root != nil {
		e.appendRows(e.Root, 0)
	}
	selected := -1
	for i := 0; i < len(e.rows); i++ {
		if e.rows[i].Node.Path == selectedPath {
			selected = i
			break
		}
	}
	if selected < 0 {
		selected = e.selected
	}
	if len(e.rows) == 0 {
		e.selected = -1
		return
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(e.rows) {
		selected = len(e.rows) - 1
	}
	e.selected = selected
}

func (e *Explorer) appendRows(node *ExplorerNode, depth int) {
	e.rows = append(e.rows, ExplorerRow{Node: node, Depth: depth})
	if !node.Directory || !node.Expanded {
		return
	}
	for i := 0; i < len(node.Children); i++ {
		e.appendRows(node.Children[i], depth+1)
	}
}

func sortExplorerNodes(nodes []*ExplorerNode) {
	for i := 1; i < len(nodes); i++ {
		value := nodes[i]
		j := i
		for j > 0 && explorerNodeLess(value, nodes[j-1]) {
			nodes[j] = nodes[j-1]
			j--
		}
		nodes[j] = value
	}
}

func explorerNodeLess(a, b *ExplorerNode) bool {
	if a.Directory != b.Directory {
		return a.Directory
	}
	folded := compareFoldASCII(a.Name, b.Name)
	if folded != 0 {
		return folded < 0
	}
	return a.Name < b.Name
}

func compareFoldASCII(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		ac := foldASCII(a[i])
		bc := foldASCII(b[i])
		if ac < bc {
			return -1
		}
		if ac > bc {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func foldASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}

func joinPath(parent, name string) string {
	if parent == "" || parent == "." {
		return name
	}
	last := parent[len(parent)-1]
	if last == '/' || last == '\\' {
		return parent + name
	}
	return parent + "/" + name
}

func trimTrailingSeparators(path string) string {
	for len(path) > 1 {
		last := path[len(path)-1]
		if last != '/' && last != '\\' {
			break
		}
		path = path[:len(path)-1]
	}
	return path
}

func pathBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
