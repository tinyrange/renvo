package forms

import (
	_ "embed"

	"renvo.dev/std/graphics"
)

// Icon identifies a vector in the embedded RENVO icon set.
type Icon int

const (
	IconNone Icon = iota
	IconNew
	IconSave
	IconBuild
	IconRun
	IconFile
	IconFolder
	IconSettings
	IconSearch
	IconCode
	IconDesigner
	IconCheck
	IconChevronRight
	IconChevronDown
	IconOpen
	IconClose
	IconDelete
	IconCopy
	IconCut
	IconPaste
	IconUndo
	IconRedo
	IconTheme
	IconMenu
	IconControlLabel
	IconControlButton
	IconControlTextBox
	IconControlTextArea
	IconControlCheckBox
	IconControlRadioButton
	IconControlPictureBox
	IconControlPanel
	IconControlComboBox
	IconControlListBox
	IconControlListView
	IconControlTreeView
	IconControlTabControl
	IconControlProgressBar
	IconControlNumericUpDown
	IconControlSlider
	IconControlGroupBox
	IconControlSplitContainer
	IconControlToolBar
	IconControlStatusBar
	IconControlMenuBar
)

const iconViewBox = 16

const (
	iconOpLine = iota + 1
	iconOpRect
	iconOpFillRect
	iconOpEllipse
	iconOpFillEllipse
	iconOpPolyline
	iconOpFillPolygon
)

type iconCommand struct {
	op     int
	values [32]int
	accent bool
}

type iconDefinition struct {
	commands []iconCommand
}

//go:embed iconset.rvi
var embeddedIconSet string

//go:embed assets/control-icons.rim
var embeddedControlIconMasks []byte

var iconDefinitions []iconDefinition
var iconDefinitionsLoaded bool
var iconDefinitionsValid bool

const controlIconSize = 64
const controlIconCount = int(IconControlMenuBar-IconControlLabel) + 1
const controlIconLayerSize = controlIconSize * controlIconSize * controlIconCount

var controlIconMasksLoaded bool
var controlIconMasksValid bool
var controlIconPrimaryMask *graphics.Image
var controlIconFillMask *graphics.Image
var controlIconAccentMask *graphics.Image

func IconCount() int { return int(IconControlMenuBar) }

func IconName(icon Icon) string {
	if icon == IconNew {
		return "new"
	} else if icon == IconSave {
		return "save"
	} else if icon == IconBuild {
		return "build"
	} else if icon == IconRun {
		return "run"
	} else if icon == IconFile {
		return "file"
	} else if icon == IconFolder {
		return "folder"
	} else if icon == IconSettings {
		return "settings"
	} else if icon == IconSearch {
		return "search"
	} else if icon == IconCode {
		return "code"
	} else if icon == IconDesigner {
		return "designer"
	} else if icon == IconCheck {
		return "check"
	} else if icon == IconChevronRight {
		return "chevron-right"
	} else if icon == IconChevronDown {
		return "chevron-down"
	} else if icon == IconOpen {
		return "open"
	} else if icon == IconClose {
		return "close"
	} else if icon == IconDelete {
		return "delete"
	} else if icon == IconCopy {
		return "copy"
	} else if icon == IconCut {
		return "cut"
	} else if icon == IconPaste {
		return "paste"
	} else if icon == IconUndo {
		return "undo"
	} else if icon == IconRedo {
		return "redo"
	} else if icon == IconTheme {
		return "theme"
	} else if icon == IconMenu {
		return "menu"
	} else if icon == IconControlLabel {
		return "control-label"
	} else if icon == IconControlButton {
		return "control-button"
	} else if icon == IconControlTextBox {
		return "control-text-box"
	} else if icon == IconControlTextArea {
		return "control-text-area"
	} else if icon == IconControlCheckBox {
		return "control-check-box"
	} else if icon == IconControlRadioButton {
		return "control-radio-button"
	} else if icon == IconControlPictureBox {
		return "control-picture-box"
	} else if icon == IconControlPanel {
		return "control-panel"
	} else if icon == IconControlComboBox {
		return "control-combo-box"
	} else if icon == IconControlListBox {
		return "control-list-box"
	} else if icon == IconControlListView {
		return "control-list-view"
	} else if icon == IconControlTreeView {
		return "control-tree-view"
	} else if icon == IconControlTabControl {
		return "control-tab-control"
	} else if icon == IconControlProgressBar {
		return "control-progress-bar"
	} else if icon == IconControlNumericUpDown {
		return "control-numeric-up-down"
	} else if icon == IconControlSlider {
		return "control-slider"
	} else if icon == IconControlGroupBox {
		return "control-group-box"
	} else if icon == IconControlSplitContainer {
		return "control-split-container"
	} else if icon == IconControlToolBar {
		return "control-tool-bar"
	} else if icon == IconControlStatusBar {
		return "control-status-bar"
	} else if icon == IconControlMenuBar {
		return "control-menu-bar"
	}
	return ""
}

func iconForName(name string) Icon {
	for icon := IconNew; icon <= IconControlMenuBar; icon++ {
		if IconName(icon) == name {
			return icon
		}
	}
	return IconNone
}

// DrawIcon renders an embedded vector into arbitrary bounds. Coordinates in
// iconset.rvi use a fixed 16x16 view box and are scaled only while drawing.
func DrawIcon(surface *graphics.Surface, icon Icon, bounds graphics.Rect, color graphics.Color) {
	DrawIconColors(surface, icon, bounds, color, color)
}

// DrawIconColors renders primary strokes and optional A-prefixed accent
// commands from the embedded vector definition with separate theme colors.
func DrawIconColors(surface *graphics.Surface, icon Icon, bounds graphics.Rect, color, accent graphics.Color) {
	if surface == nil || icon <= IconNone || int(icon) > IconCount() || bounds.Empty() {
		return
	}
	if icon >= IconControlLabel {
		DrawControlIcon(surface, icon, bounds, color, graphics.Color{}, accent)
		return
	}
	loadIconDefinitions()
	if !iconDefinitionsValid || int(icon) >= len(iconDefinitions) {
		return
	}
	commands := iconDefinitions[int(icon)].commands
	scaleX := bounds.Width() / iconViewBox
	scaleY := bounds.Height() / iconViewBox
	strokeScale := scaleX
	if scaleY < strokeScale {
		strokeScale = scaleY
	}
	for i := 0; i < len(commands); i++ {
		command := commands[i]
		commandColor := color
		if command.accent {
			commandColor = accent
		}
		if command.op == iconOpLine {
			surface.DrawLine(iconPoint(bounds, scaleX, scaleY, command.values[0], command.values[1]), iconPoint(bounds, scaleX, scaleY, command.values[2], command.values[3]), graphics.Scalar(command.values[4])*strokeScale, commandColor)
		} else if command.op == iconOpRect || command.op == iconOpFillRect {
			rect := iconRect(bounds, scaleX, scaleY, command.values[0], command.values[1], command.values[2], command.values[3])
			if command.op == iconOpRect {
				surface.StrokeRect(rect, graphics.Scalar(command.values[4])*strokeScale, commandColor)
			} else {
				surface.FillRect(rect, commandColor)
			}
		} else if command.op == iconOpEllipse || command.op == iconOpFillEllipse {
			rect := iconRect(bounds, scaleX, scaleY, command.values[0], command.values[1], command.values[2], command.values[3])
			if command.op == iconOpEllipse {
				surface.StrokeEllipse(rect, graphics.Scalar(command.values[4])*strokeScale, commandColor)
			} else {
				surface.FillEllipse(rect, commandColor)
			}
		} else if command.op == iconOpPolyline {
			count := command.values[1]
			for point := 1; point < count; point++ {
				from := 2 + (point-1)*2
				to := 2 + point*2
				surface.DrawLine(iconPoint(bounds, scaleX, scaleY, command.values[from], command.values[from+1]), iconPoint(bounds, scaleX, scaleY, command.values[to], command.values[to+1]), graphics.Scalar(command.values[0])*strokeScale, commandColor)
			}
		} else if command.op == iconOpFillPolygon {
			count := command.values[0]
			points := make([]graphics.Point, count)
			for point := 0; point < count; point++ {
				at := 1 + point*2
				points[point] = iconPoint(bounds, scaleX, scaleY, command.values[at], command.values[at+1])
			}
			surface.FillConvexPolygon(points, commandColor)
		}
	}
}

// DrawControlIcon renders the generated raster artwork for a designer control.
// The embedded image is stored as primary, fill, and accent alpha masks so the
// original imagegen shapes remain crisp while still following application
// themes instead of baking light-theme colors into the binary.
func DrawControlIcon(surface *graphics.Surface, icon Icon, bounds graphics.Rect, primary, fill, accent graphics.Color) {
	if surface == nil || icon < IconControlLabel || icon > IconControlMenuBar || bounds.Empty() {
		return
	}
	loadControlIconMasks()
	if !controlIconMasksValid {
		return
	}
	index := int(icon - IconControlLabel)
	source := graphics.R(graphics.Scalar(index*controlIconSize), 0, controlIconSize, controlIconSize)
	if fill.A != 0 {
		surface.DrawImage(controlIconFillMask, source, bounds, graphics.SamplingLinear, fill)
	}
	if primary.A != 0 {
		surface.DrawImage(controlIconPrimaryMask, source, bounds, graphics.SamplingLinear, primary)
	}
	if accent.A != 0 {
		surface.DrawImage(controlIconAccentMask, source, bounds, graphics.SamplingLinear, accent)
	}
}

func loadControlIconMasks() {
	if controlIconMasksLoaded {
		return
	}
	controlIconMasksLoaded = true
	headerSize := 8
	expectedSize := headerSize + controlIconLayerSize*3
	if len(embeddedControlIconMasks) != expectedSize || embeddedControlIconMasks[0] != 'R' || embeddedControlIconMasks[1] != 'I' || embeddedControlIconMasks[2] != 'M' || embeddedControlIconMasks[3] != '1' || int(embeddedControlIconMasks[4]) != controlIconSize || int(embeddedControlIconMasks[5]) != controlIconCount || embeddedControlIconMasks[6] != 3 {
		return
	}
	primaryStart := headerSize
	fillStart := primaryStart + controlIconLayerSize
	accentStart := fillStart + controlIconLayerSize
	width := controlIconSize * controlIconCount
	controlIconPrimaryMask = graphics.NewMask(width, controlIconSize, embeddedControlIconMasks[primaryStart:fillStart])
	controlIconFillMask = graphics.NewMask(width, controlIconSize, embeddedControlIconMasks[fillStart:accentStart])
	controlIconAccentMask = graphics.NewMask(width, controlIconSize, embeddedControlIconMasks[accentStart:])
	controlIconMasksValid = true
}

func drawIcon(surface *graphics.Surface, icon Icon, x, y graphics.Scalar, color graphics.Color) {
	DrawIcon(surface, icon, graphics.R(x, y, iconViewBox, iconViewBox), color)
}

func drawChevronIcon(surface *graphics.Surface, x, y graphics.Scalar, expanded bool, color graphics.Color) {
	icon := IconChevronRight
	if expanded {
		icon = IconChevronDown
	}
	DrawIcon(surface, icon, graphics.R(x, y, 9, 9), color)
}

func iconPoint(bounds graphics.Rect, scaleX, scaleY graphics.Scalar, x, y int) graphics.Point {
	return graphics.Point{X: bounds.MinX + graphics.Scalar(x)*scaleX, Y: bounds.MinY + graphics.Scalar(y)*scaleY}
}

func iconRect(bounds graphics.Rect, scaleX, scaleY graphics.Scalar, x, y, width, height int) graphics.Rect {
	return graphics.R(bounds.MinX+graphics.Scalar(x)*scaleX, bounds.MinY+graphics.Scalar(y)*scaleY, graphics.Scalar(width)*scaleX, graphics.Scalar(height)*scaleY)
}

func loadIconDefinitions() {
	if iconDefinitionsLoaded {
		return
	}
	iconDefinitionsLoaded = true
	iconDefinitions = make([]iconDefinition, IconCount()+1)
	tokens := iconTokens(embeddedIconSet)
	at := 0
	current := IconNone
	valid := true
	for at < len(tokens) && valid {
		op := tokens[at]
		at++
		if op == "I" {
			if at >= len(tokens) {
				valid = false
				break
			}
			current = iconForName(tokens[at])
			at++
			valid = current != IconNone
			continue
		}
		if current == IconNone {
			valid = false
			break
		}
		command := iconCommand{}
		if len(op) == 2 && op[0] == 'A' {
			command.accent = true
			op = op[1:]
		}
		valueCount := 0
		if op == "L" {
			command.op, valueCount = iconOpLine, 5
		} else if op == "R" {
			command.op, valueCount = iconOpRect, 5
		} else if op == "F" {
			command.op, valueCount = iconOpFillRect, 4
		} else if op == "O" {
			command.op, valueCount = iconOpEllipse, 5
		} else if op == "E" {
			command.op, valueCount = iconOpFillEllipse, 4
		} else if op == "P" || op == "G" {
			width := 0
			if op == "P" {
				command.op = iconOpPolyline
				width, valid = iconReadInt(tokens, &at)
			} else {
				command.op = iconOpFillPolygon
			}
			count := 0
			if valid {
				count, valid = iconReadInt(tokens, &at)
			}
			if !valid || count < 2 || count > 15 {
				valid = false
				break
			}
			valueAt := 0
			if op == "P" {
				command.values[0] = width
				command.values[1] = count
				valueAt = 2
			} else {
				command.values[0] = count
				valueAt = 1
			}
			for value := 0; value < count*2 && valid; value++ {
				command.values[valueAt+value], valid = iconReadInt(tokens, &at)
			}
		} else {
			valid = false
			break
		}
		for value := 0; value < valueCount && valid; value++ {
			command.values[value], valid = iconReadInt(tokens, &at)
		}
		if valid {
			definition := &iconDefinitions[int(current)]
			definition.commands = append(definition.commands, command)
		}
	}
	if valid {
		for icon := IconNew; icon <= IconControlMenuBar; icon++ {
			if len(iconDefinitions[int(icon)].commands) == 0 {
				valid = false
			}
		}
	}
	iconDefinitionsValid = valid
}

func iconTokens(text string) []string {
	tokens := []string{}
	start := -1
	comment := false
	for i := 0; i <= len(text); i++ {
		ch := byte(' ')
		if i < len(text) {
			ch = text[i]
		}
		if comment {
			if ch == '\n' {
				comment = false
			}
			continue
		}
		if ch == '#' {
			if start >= 0 {
				tokens = append(tokens, text[start:i])
				start = -1
			}
			comment = true
			continue
		}
		space := ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
		if space {
			if start >= 0 {
				tokens = append(tokens, text[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	return tokens
}

func iconReadInt(tokens []string, at *int) (int, bool) {
	if *at >= len(tokens) {
		return 0, false
	}
	text := tokens[*at]
	*at++
	if text == "" {
		return 0, false
	}
	sign := 1
	index := 0
	if text[0] == '-' {
		sign = -1
		index = 1
	}
	if index >= len(text) {
		return 0, false
	}
	value := 0
	for ; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return 0, false
		}
		value = value*10 + int(text[index]-'0')
	}
	return value * sign, true
}
