package forms

import "renvo.dev/std/graphics"

const menuBarHorizontalPadding = 12
const menuBarMinimumItemWidth = 58
const menuItemHorizontalPadding = 12
const menuItemHeight = 28
const menuSeparatorHeight = 9
const menuDropPadding = 4
const menuDropMinimumWidth = 240

// Shortcut describes a portable menu shortcut. Primary accepts Command on
// macOS and Control on the other desktop hosts, while Text controls only the
// label displayed in the menu.
type Shortcut struct {
	Key       graphics.Key
	Modifiers graphics.Modifiers
	Primary   bool
	Text      string
}

type MenuItem struct {
	text      string
	enabled   bool
	separator bool
	checkable bool
	checked   bool
	icon      Icon
	shortcut  Shortcut
	menu      *Menu
	Activate  EventHandler
}

func NewMenuItem(text string) *MenuItem {
	return &MenuItem{text: text, enabled: true}
}

func NewMenuSeparator() *MenuItem {
	return &MenuItem{enabled: false, separator: true}
}

func (item *MenuItem) Text() string       { return item.text }
func (item *MenuItem) Enabled() bool      { return item.enabled }
func (item *MenuItem) Shortcut() Shortcut { return item.shortcut }
func (item *MenuItem) Icon() Icon         { return item.icon }
func (item *MenuItem) Checkable() bool    { return item.checkable }
func (item *MenuItem) Checked() bool      { return item.checked }

func (item *MenuItem) SetText(text string) {
	if item == nil || item.text == text {
		return
	}
	item.text = text
	item.invalidate()
}

func (item *MenuItem) SetEnabled(enabled bool) {
	if item == nil || item.enabled == enabled || item.separator {
		return
	}
	item.enabled = enabled
	item.invalidate()
}

func (item *MenuItem) SetShortcut(shortcut Shortcut) {
	if item == nil || item.shortcut == shortcut {
		return
	}
	item.shortcut = shortcut
	item.invalidate()
}

func (item *MenuItem) SetIcon(icon Icon) {
	if item == nil || item.icon == icon {
		return
	}
	item.icon = icon
	item.invalidate()
}

func (item *MenuItem) SetChecked(checked bool) {
	if item == nil || item.separator || item.checkable && item.checked == checked {
		return
	}
	item.checkable = true
	item.checked = checked
	item.invalidate()
}

func (item *MenuItem) invalidate() {
	if item != nil && item.menu != nil && item.menu.bar != nil {
		item.menu.bar.refreshBounds()
		item.menu.bar.AccessibilityChildrenChanged()
		item.menu.bar.Invalidate()
	}
}

type Menu struct {
	text  string
	items []*MenuItem
	bar   *MenuBar
}

func NewMenu(text string) *Menu { return &Menu{text: text} }
func (menu *Menu) Text() string { return menu.text }

func (menu *Menu) SetText(text string) {
	if menu == nil || menu.text == text {
		return
	}
	menu.text = text
	if menu.bar != nil {
		menu.bar.refreshBounds()
		menu.bar.AccessibilityChildrenChanged()
		menu.bar.Invalidate()
	}
}

func (menu *Menu) Add(item *MenuItem) {
	if menu == nil || item == nil || item.menu == menu {
		return
	}
	if item.menu != nil {
		return
	}
	item.menu = menu
	menu.items = append(menu.items, item)
	if menu.bar != nil {
		menu.bar.refreshBounds()
		menu.bar.AccessibilityChildrenChanged()
		menu.bar.Invalidate()
	}
}

// MenuBar is a retained, generated-code-friendly application menu. It stays
// out of the tab order and receives command keys before the focused editor.
type MenuBar struct {
	Control
	font         *graphics.Font
	menus        []*Menu
	barBounds    graphics.Rect
	openMenu     int
	selectedItem int
	hoverMenu    int
}

func NewMenuBar() *MenuBar {
	bar := &MenuBar{openMenu: -1, selectedItem: -1, hoverMenu: -1}
	bar.Control = *NewControl()
	bar.Control.applyTheme = bar.applyTheme
	bar.Control.layoutBounds = bar.setLayoutBounds
	bar.SetTabStop(false)
	bar.SetAccessibilityRole(AccessibilityRoleMenuBar)
	bar.SetAccessibilityName("Application menu")
	bar.AccessibilityChildren = bar.accessibilityChildren
	bar.AccessibilityPerform = bar.accessibilityPerform
	bar.applyTheme(LightTheme())
	bar.Paint = bar.paint
	bar.PointerDown = bar.pointerDown
	bar.PointerMove = bar.pointerMove
	bar.PointerLeave = bar.pointerLeave
	bar.Control.dismiss = bar.dismiss
	return bar
}

func (bar *MenuBar) applyTheme(theme Theme) { applyRaisedTheme(&bar.Control, theme) }

func (bar *MenuBar) Font() *graphics.Font { return bar.font }
func (bar *MenuBar) Open() bool           { return bar != nil && bar.openMenu >= 0 }

func (bar *MenuBar) SetFont(font *graphics.Font) {
	if bar == nil || bar.font == font {
		return
	}
	bar.font = font
	bar.refreshBounds()
	bar.Invalidate()
}

func (bar *MenuBar) SetBounds(bounds graphics.Rect) {
	if bar == nil {
		return
	}
	bar.Control.setPreferredBounds(bounds)
	if bar.Form() != nil && bar.Dock() != DockNone {
		bar.Form().performLayout()
		return
	}
	bar.setLayoutBounds(bounds)
}

func (bar *MenuBar) setLayoutBounds(bounds graphics.Rect) {
	if bar == nil {
		return
	}
	bar.barBounds = bounds
	bar.refreshBounds()
}

func (bar *MenuBar) Add(menu *Menu) {
	if bar == nil || menu == nil || menu.bar == bar {
		return
	}
	if menu.bar != nil {
		return
	}
	menu.bar = bar
	bar.menus = append(bar.menus, menu)
	bar.refreshBounds()
	bar.AccessibilityChildrenChanged()
	bar.Invalidate()
}

func (bar *MenuBar) refreshBounds() {
	if bar == nil {
		return
	}
	bounds := bar.barBounds
	if bar.openMenu >= 0 && bar.openMenu < len(bar.menus) {
		x := bar.menuX(bar.openMenu)
		width := bar.dropWidth(bar.menus[bar.openMenu])
		height := bar.dropHeight(bar.menus[bar.openMenu])
		if x+width > bounds.MaxX {
			bounds.MaxX = x + width
		}
		bounds.MaxY = bar.barBounds.MaxY + height
	}
	bar.Control.setBoundsCore(bounds)
}

func (bar *MenuBar) dismiss() {
	if bar == nil || bar.openMenu < 0 {
		return
	}
	bar.openMenu = -1
	bar.selectedItem = -1
	bar.Control.popup = false
	bar.refreshBounds()
	bar.AccessibilityChildrenChanged()
	bar.Invalidate()
}

func (bar *MenuBar) menuWidth(menu *Menu) graphics.Scalar {
	if menu == nil {
		return 0
	}
	width := graphics.Scalar(len(menu.text)*8) + menuBarHorizontalPadding*2
	if bar.font != nil {
		width = graphics.MeasureText(bar.font, menu.text).Width + menuBarHorizontalPadding*2
	}
	if width < menuBarMinimumItemWidth {
		width = menuBarMinimumItemWidth
	}
	return width
}

func (bar *MenuBar) menuX(index int) graphics.Scalar {
	x := bar.barBounds.MinX
	for i := 0; i < index && i < len(bar.menus); i++ {
		x += bar.menuWidth(bar.menus[i])
	}
	return x
}

func (bar *MenuBar) menuAt(x, y graphics.Scalar) int {
	if y < bar.barBounds.MinY || y >= bar.barBounds.MaxY {
		return -1
	}
	for i := 0; i < len(bar.menus); i++ {
		start := bar.menuX(i)
		if x >= start && x < start+bar.menuWidth(bar.menus[i]) {
			return i
		}
	}
	return -1
}

func (bar *MenuBar) dropWidth(menu *Menu) graphics.Scalar {
	width := graphics.Scalar(menuDropMinimumWidth)
	if menu == nil {
		return width
	}
	for i := 0; i < len(menu.items); i++ {
		item := menu.items[i]
		if item.separator {
			continue
		}
		textWidth := graphics.Scalar(len(item.text) * 8)
		shortcutWidth := graphics.Scalar(len(item.shortcut.Text) * 8)
		if bar.font != nil {
			textWidth = graphics.MeasureText(bar.font, item.text).Width
			shortcutWidth = graphics.MeasureText(bar.font, item.shortcut.Text).Width
		}
		candidate := textWidth + shortcutWidth + menuItemHorizontalPadding*2
		if item.icon != IconNone || item.checkable {
			candidate += 24
		}
		if item.shortcut.Text != "" {
			candidate += 28
		}
		if candidate > width {
			width = candidate
		}
	}
	return width
}

func (bar *MenuBar) dropHeight(menu *Menu) graphics.Scalar {
	height := graphics.Scalar(menuDropPadding * 2)
	if menu == nil {
		return height
	}
	for i := 0; i < len(menu.items); i++ {
		if menu.items[i].separator {
			height += menuSeparatorHeight
		} else {
			height += menuItemHeight
		}
	}
	return height
}

func (bar *MenuBar) itemAt(y graphics.Scalar) int {
	if bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return -1
	}
	menu := bar.menus[bar.openMenu]
	current := bar.barBounds.MaxY + menuDropPadding
	for i := 0; i < len(menu.items); i++ {
		height := graphics.Scalar(menuItemHeight)
		if menu.items[i].separator {
			height = menuSeparatorHeight
		}
		if y >= current && y < current+height {
			if menu.items[i].separator {
				return -1
			}
			return i
		}
		current += height
	}
	return -1
}

func (bar *MenuBar) pointerDown(x, y graphics.Scalar) {
	globalX := x + bar.Bounds().MinX
	globalY := y + bar.Bounds().MinY
	menuIndex := bar.menuAt(globalX, globalY)
	if menuIndex >= 0 {
		if bar.openMenu == menuIndex {
			bar.dismiss()
			return
		}
		bar.openMenu = menuIndex
		bar.selectedItem = bar.firstEnabledItem(bar.menus[menuIndex])
		bar.Control.popup = true
		bar.refreshBounds()
		bar.AccessibilityChildrenChanged()
		bar.Invalidate()
		return
	}
	if bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return
	}
	dropX := bar.menuX(bar.openMenu)
	if globalX < dropX || globalX >= dropX+bar.dropWidth(bar.menus[bar.openMenu]) {
		bar.dismiss()
		return
	}
	index := bar.itemAt(globalY)
	if index >= 0 {
		bar.activate(index)
		return
	}
	bar.dismiss()
}

func (bar *MenuBar) pointerMove(x, y graphics.Scalar) {
	globalX := x + bar.Bounds().MinX
	globalY := y + bar.Bounds().MinY
	menuIndex := bar.menuAt(globalX, globalY)
	if bar.openMenu < 0 {
		if bar.hoverMenu != menuIndex {
			bar.hoverMenu = menuIndex
			bar.Invalidate()
		}
		return
	}
	if menuIndex >= 0 && menuIndex != bar.openMenu {
		bar.openMenu = menuIndex
		bar.selectedItem = bar.firstEnabledItem(bar.menus[menuIndex])
		bar.refreshBounds()
		bar.AccessibilityChildrenChanged()
		bar.Invalidate()
		return
	}
	dropX := bar.menuX(bar.openMenu)
	drop := graphics.R(dropX, bar.barBounds.MaxY, bar.dropWidth(bar.menus[bar.openMenu]), bar.dropHeight(bar.menus[bar.openMenu]))
	if menuIndex < 0 && !pointInRect(globalX, globalY, drop) {
		bar.dismiss()
		return
	}
	index := bar.itemAt(globalY)
	if index >= 0 && bar.menus[bar.openMenu].items[index].enabled && index != bar.selectedItem {
		bar.selectedItem = index
		bar.AccessibilityChildrenChanged()
		bar.Invalidate()
		return
	}
}

func (bar *MenuBar) pointerLeave() {
	bar.hoverMenu = -1
	if bar.openMenu >= 0 {
		bar.dismiss()
	} else {
		bar.Invalidate()
	}
}

func (bar *MenuBar) activate(index int) {
	if bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return
	}
	menu := bar.menus[bar.openMenu]
	if index < 0 || index >= len(menu.items) || !menu.items[index].enabled || menu.items[index].separator {
		return
	}
	item := menu.items[index]
	bar.dismiss()
	if item.Activate != nil {
		item.Activate()
	}
}

func (bar *MenuBar) commandKey(event graphics.Event) bool {
	if bar == nil {
		return false
	}
	if bar.openMenu >= 0 {
		if event.Key == graphics.KeyTab {
			bar.dismiss()
			return false
		}
		if event.Key == graphics.KeyEscape {
			bar.dismiss()
			return true
		}
		if event.Key == graphics.KeyLeft || event.Key == graphics.KeyRight {
			direction := -1
			if event.Key == graphics.KeyRight {
				direction = 1
			}
			bar.openMenu = (bar.openMenu + direction + len(bar.menus)) % len(bar.menus)
			bar.selectedItem = bar.firstEnabledItem(bar.menus[bar.openMenu])
			bar.refreshBounds()
			bar.AccessibilityChildrenChanged()
			bar.Invalidate()
			return true
		}
		if event.Key == graphics.KeyUp || event.Key == graphics.KeyDown {
			bar.moveSelection(event.Key == graphics.KeyDown)
			return true
		}
		if event.Key == graphics.KeyHome {
			bar.selectedItem = bar.firstEnabledItem(bar.menus[bar.openMenu])
			bar.AccessibilityChildrenChanged()
			bar.Invalidate()
			return true
		}
		if event.Key == graphics.KeyEnd {
			bar.selectedItem = bar.lastEnabledItem(bar.menus[bar.openMenu])
			bar.AccessibilityChildrenChanged()
			bar.Invalidate()
			return true
		}
		if event.Key == graphics.KeyEnter || event.Key == graphics.KeySpace {
			bar.activate(bar.selectedItem)
			return true
		}
	}
	for i := 0; i < len(bar.menus); i++ {
		for j := 0; j < len(bar.menus[i].items); j++ {
			item := bar.menus[i].items[j]
			if item.enabled && !item.separator && shortcutMatches(item.shortcut, event) {
				if item.Activate != nil {
					item.Activate()
				}
				return true
			}
		}
	}
	return false
}

func (bar *MenuBar) commandText(value string) bool {
	if bar == nil || bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return false
	}
	items := bar.menus[bar.openMenu].items
	if value == "" || len(items) == 0 {
		return true
	}
	start := bar.selectedItem
	for count := 0; count < len(items); count++ {
		index := (start + count + 1 + len(items)) % len(items)
		item := items[index]
		if item.enabled && !item.separator && menuTextStartsWith(item.text, value) {
			bar.selectedItem = index
			bar.AccessibilityChildrenChanged()
			bar.Invalidate()
			return true
		}
	}
	return true
}

func menuTextStartsWith(text, prefix string) bool {
	if prefix == "" || len(text) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		left := text[i]
		right := prefix[i]
		if left >= 'A' && left <= 'Z' {
			left += 'a' - 'A'
		}
		if right >= 'A' && right <= 'Z' {
			right += 'a' - 'A'
		}
		if left != right {
			return false
		}
	}
	return true
}

func (bar *MenuBar) lastEnabledItem(menu *Menu) int {
	if menu == nil {
		return -1
	}
	for i := len(menu.items) - 1; i >= 0; i-- {
		if menu.items[i].enabled && !menu.items[i].separator {
			return i
		}
	}
	return -1
}

func (bar *MenuBar) firstEnabledItem(menu *Menu) int {
	if menu == nil {
		return -1
	}
	for i := 0; i < len(menu.items); i++ {
		if menu.items[i].enabled && !menu.items[i].separator {
			return i
		}
	}
	return -1
}

func (bar *MenuBar) moveSelection(forward bool) {
	if bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return
	}
	items := bar.menus[bar.openMenu].items
	if len(items) == 0 {
		return
	}
	direction := -1
	if forward {
		direction = 1
	}
	index := bar.selectedItem
	for count := 0; count < len(items); count++ {
		index = (index + direction + len(items)) % len(items)
		if items[index].enabled && !items[index].separator {
			bar.selectedItem = index
			bar.AccessibilityChildrenChanged()
			bar.Invalidate()
			return
		}
	}
}

func (bar *MenuBar) accessibilityChildren() []AccessibilityNode {
	if bar == nil {
		return nil
	}
	nodes := make([]AccessibilityNode, 0, len(bar.menus)+8)
	baseID := bar.AccessibilityID()
	for i := 0; i < len(bar.menus); i++ {
		menuID := baseID + "-menu-" + accessibilityDecimal(i+1)
		nodes = append(nodes, AccessibilityNode{
			ID:         menuID,
			Role:       AccessibilityRoleMenuItem,
			Name:       bar.menus[i].text,
			Bounds:     graphics.R(bar.menuX(i), bar.barBounds.MinY, bar.menuWidth(bar.menus[i]), bar.barBounds.Height()),
			Actions:    AccessibilitySupportsInvoke,
			Selectable: true,
			Selected:   i == bar.openMenu,
		})
		if i != bar.openMenu {
			continue
		}
		y := bar.barBounds.MaxY + menuDropPadding
		width := bar.dropWidth(bar.menus[i])
		for j := 0; j < len(bar.menus[i].items); j++ {
			item := bar.menus[i].items[j]
			height := graphics.Scalar(menuItemHeight)
			role := AccessibilityRoleMenuItem
			if item.separator {
				height = menuSeparatorHeight
				role = AccessibilityRoleSeparator
			}
			actions := 0
			if item.enabled && !item.separator {
				actions = AccessibilitySupportsInvoke
			}
			nodes = append(nodes, AccessibilityNode{
				ID:          menuID + "-item-" + accessibilityDecimal(j+1),
				ParentID:    menuID,
				Role:        role,
				Name:        item.text,
				Description: item.shortcut.Text,
				Bounds:      graphics.R(bar.menuX(i), y, width, height),
				Actions:     actions,
				Disabled:    !item.enabled,
				Selectable:  !item.separator,
				Selected:    j == bar.selectedItem,
				Checkable:   item.checkable,
				Checked:     item.checked,
			})
			y += height
		}
	}
	return nodes
}

func (bar *MenuBar) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	if bar == nil || action != AccessibilityActionInvoke {
		return false
	}
	menuIndex, itemIndex, ok := menuAccessibilityIndices(id, bar.AccessibilityID()+"-menu-")
	if !ok || menuIndex < 0 || menuIndex >= len(bar.menus) {
		return false
	}
	if itemIndex < 0 {
		if bar.openMenu == menuIndex {
			bar.dismiss()
		} else {
			bar.openMenu = menuIndex
			bar.selectedItem = bar.firstEnabledItem(bar.menus[menuIndex])
			bar.Control.popup = true
			bar.refreshBounds()
			bar.Invalidate()
		}
		return true
	}
	if bar.openMenu != menuIndex {
		return false
	}
	bar.activate(itemIndex)
	return true
}

func menuAccessibilityIndices(id, prefix string) (int, int, bool) {
	if len(id) <= len(prefix) || !accessibilityHasPrefix(id, prefix) {
		return -1, -1, false
	}
	at := len(prefix)
	menu := 0
	for at < len(id) && id[at] >= '0' && id[at] <= '9' {
		menu = menu*10 + int(id[at]-'0')
		at++
	}
	if menu == 0 {
		return -1, -1, false
	}
	if at == len(id) {
		return menu - 1, -1, true
	}
	itemPrefix := "-item-"
	if !accessibilityHasPrefix(id[at:], itemPrefix) {
		return -1, -1, false
	}
	at += len(itemPrefix)
	item := 0
	for at < len(id) && id[at] >= '0' && id[at] <= '9' {
		item = item*10 + int(id[at]-'0')
		at++
	}
	return menu - 1, item - 1, item > 0 && at == len(id)
}

func accessibilityHasPrefix(text, prefix string) bool {
	if len(text) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if text[i] != prefix[i] {
			return false
		}
	}
	return true
}

func shortcutMatches(shortcut Shortcut, event graphics.Event) bool {
	if shortcut.Key == graphics.KeyUnknown || shortcut.Key != event.Key {
		return false
	}
	actual := event.Modifiers
	if shortcut.Primary {
		primary := actual&graphics.ModifierCommand != 0 || actual&graphics.ModifierControl != 0 && actual&graphics.ModifierAlt == 0
		if !primary {
			return false
		}
		actual &^= graphics.ModifierCommand | graphics.ModifierControl
	}
	return actual == shortcut.Modifiers
}

func (bar *MenuBar) paint(surface *graphics.Surface) {
	if bar == nil {
		return
	}
	theme := controlTheme(&bar.Control)
	surface.FillRect(bar.barBounds, bar.Background())
	for i := 0; i < len(bar.menus); i++ {
		x := bar.menuX(i)
		itemBounds := graphics.R(x, bar.barBounds.MinY, bar.menuWidth(bar.menus[i]), bar.barBounds.Height())
		if i == bar.openMenu {
			surface.FillRect(itemBounds, theme.Selection)
		} else if i == bar.hoverMenu {
			surface.FillRect(itemBounds, theme.Hover)
		}
		bar.drawText(surface, itemBounds.MinX+menuBarHorizontalPadding, centeredBaseline(bar.font, itemBounds), bar.menus[i].text, controlForeground(&bar.Control))
	}
	if bar.openMenu < 0 || bar.openMenu >= len(bar.menus) {
		return
	}
	menu := bar.menus[bar.openMenu]
	drop := graphics.R(bar.menuX(bar.openMenu), bar.barBounds.MaxY, bar.dropWidth(menu), bar.dropHeight(menu))
	surface.FillRect(drop, theme.Surface)
	surface.StrokeRect(drop, 1, theme.Border)
	y := drop.MinY + menuDropPadding
	for i := 0; i < len(menu.items); i++ {
		item := menu.items[i]
		if item.separator {
			surface.DrawLine(graphics.Point{X: drop.MinX + 7, Y: y + menuSeparatorHeight/2}, graphics.Point{X: drop.MaxX - 7, Y: y + menuSeparatorHeight/2}, 1, theme.Border)
			y += menuSeparatorHeight
			continue
		}
		row := graphics.R(drop.MinX+2, y, drop.Width()-4, menuItemHeight)
		if i == bar.selectedItem && item.enabled {
			surface.FillRect(row, theme.Selection)
		}
		color := theme.Text
		if !item.enabled {
			color = theme.Disabled
		}
		baseline := centeredBaseline(bar.font, row)
		textX := row.MinX + menuItemHorizontalPadding
		icon := item.icon
		if item.checked {
			icon = IconCheck
		}
		if icon != IconNone {
			drawIcon(surface, icon, textX, row.MinY+(row.Height()-15)/2, color)
		}
		if icon != IconNone || item.checkable {
			textX += 24
		}
		bar.drawText(surface, textX, baseline, item.text, color)
		if item.shortcut.Text != "" {
			width := graphics.Scalar(len(item.shortcut.Text) * 8)
			if bar.font != nil {
				width = graphics.MeasureText(bar.font, item.shortcut.Text).Width
			}
			bar.drawText(surface, row.MaxX-menuItemHorizontalPadding-width, baseline, item.shortcut.Text, color)
		}
		y += menuItemHeight
	}
}

func (bar *MenuBar) drawText(surface *graphics.Surface, x, baseline graphics.Scalar, text string, color graphics.Color) {
	if bar.font != nil {
		surface.DrawText(bar.font, graphics.Point{X: x, Y: baseline}, text, color)
	}
}

func centeredBaseline(font *graphics.Font, bounds graphics.Rect) graphics.Scalar {
	if font == nil {
		return bounds.MinY
	}
	height := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	return bounds.MinY + (bounds.Height()-height)/2 + font.Metrics.Ascent
}
