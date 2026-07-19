package ide

const editorTabWidth = 4

type LineEnding int

const (
	LineEndingLF LineEnding = iota
	LineEndingCRLF
)

type editKind int

const (
	editReplace editKind = iota
	editInsert
	editBackspace
	editDelete
)

type documentEdit struct {
	start        int
	before       []byte
	after        []byte
	anchorBefore int
	caretBefore  int
	anchorAfter  int
	caretAfter   int
	kind         editKind
}

// Document is the platform-neutral state of one editable UTF-8 file. Offsets
// are byte offsets into the normalized LF text and are always kept on UTF-8
// boundaries. Bytes converts back to the document's original line-ending
// convention for saving.
type Document struct {
	text            []byte
	lineStarts      []int
	Anchor          int
	Caret           int
	desiredColumn   int
	desiredValid    bool
	lineEnding      LineEnding
	history         []documentEdit
	historyPosition int
	savedHistory    int
}

func NewDocument(data []byte) *Document {
	d := &Document{}
	d.Reset(data)
	return d
}

func (d *Document) Reset(data []byte) {
	if d == nil {
		return
	}
	d.lineEnding = detectLineEnding(data)
	d.text = normalizeLineEndings(data)
	d.Anchor = 0
	d.Caret = 0
	d.desiredValid = false
	d.history = nil
	d.historyPosition = 0
	d.savedHistory = 0
	d.rebuildLines()
}

func (d *Document) Text() string {
	if d == nil {
		return ""
	}
	return string(d.text)
}

func (d *Document) Bytes() []byte {
	if d == nil {
		return nil
	}
	if d.lineEnding == LineEndingLF {
		out := make([]byte, len(d.text))
		copy(out, d.text)
		return out
	}
	out := make([]byte, 0, len(d.text)+d.LineCount())
	for i := 0; i < len(d.text); i++ {
		if d.text[i] == '\n' {
			out = append(out, '\r')
		}
		out = append(out, d.text[i])
	}
	return out
}

func (d *Document) LineEnding() LineEnding {
	if d == nil {
		return LineEndingLF
	}
	return d.lineEnding
}

func (d *Document) SetLineEnding(ending LineEnding) {
	if d != nil {
		d.lineEnding = ending
	}
}

func (d *Document) Dirty() bool {
	return d != nil && d.historyPosition != d.savedHistory
}

func (d *Document) MarkSaved() {
	if d != nil {
		d.savedHistory = d.historyPosition
	}
}

func (d *Document) LineCount() int {
	if d == nil {
		return 0
	}
	return len(d.lineStarts)
}

func (d *Document) LineText(line int) string {
	if d == nil || line < 0 || line >= len(d.lineStarts) {
		return ""
	}
	start := d.lineStarts[line]
	end := d.lineEnd(line)
	return string(d.text[start:end])
}

// Position returns the zero-based logical line and rune column for offset.
func (d *Document) Position(offset int) (int, int) {
	if d == nil || len(d.lineStarts) == 0 {
		return 0, 0
	}
	offset = d.clampOffset(offset)
	line := d.lineForOffset(offset)
	return line, runeCount(d.text[d.lineStarts[line]:offset])
}

// VisualPosition returns the line and rendered cell column for an offset.
// Tabs occupy the same four cells used by graphics.DrawText; all other UTF-8
// runes occupy one editor cell.
func (d *Document) VisualPosition(offset int) (int, int) {
	if d == nil || len(d.lineStarts) == 0 {
		return 0, 0
	}
	offset = d.clampOffset(offset)
	line := d.lineForOffset(offset)
	return line, visualColumn(d.text, d.lineStarts[line], offset)
}

// VisualOffset maps a rendered cell column back to the nearest valid UTF-8
// boundary on a line.
func (d *Document) VisualOffset(line, column int) int {
	if d == nil || len(d.lineStarts) == 0 {
		return 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(d.lineStarts) {
		line = len(d.lineStarts) - 1
	}
	if column < 0 {
		column = 0
	}
	return offsetForVisualColumn(d.text, d.lineStarts[line], d.lineEnd(line), column)
}

// Offset returns the byte offset for a zero-based line and rune column.
// Columns beyond the line clamp to its end.
func (d *Document) Offset(line, column int) int {
	if d == nil || len(d.lineStarts) == 0 {
		return 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(d.lineStarts) {
		line = len(d.lineStarts) - 1
	}
	if column < 0 {
		column = 0
	}
	start := d.lineStarts[line]
	end := d.lineEnd(line)
	return offsetForRuneColumn(d.text, start, end, column)
}

func (d *Document) SetSelection(anchor, caret int) {
	if d == nil {
		return
	}
	d.Anchor = d.clampOffset(anchor)
	d.Caret = d.clampOffset(caret)
	d.desiredValid = false
}

func (d *Document) SelectAll() {
	if d == nil {
		return
	}
	d.Anchor = 0
	d.Caret = len(d.text)
	d.desiredValid = false
}

func (d *Document) Selection() (int, int) {
	if d == nil {
		return 0, 0
	}
	if d.Anchor <= d.Caret {
		return d.Anchor, d.Caret
	}
	return d.Caret, d.Anchor
}

func (d *Document) SelectedText() string {
	start, end := d.Selection()
	if d == nil || start == end {
		return ""
	}
	return string(d.text[start:end])
}

func (d *Document) MoveCharacter(delta int, extend bool) {
	if d == nil || delta == 0 {
		return
	}
	start, end := d.Selection()
	if !extend && start != end {
		if delta < 0 {
			d.moveCaret(start, false)
		} else {
			d.moveCaret(end, false)
		}
		return
	}
	offset := d.Caret
	if delta < 0 {
		for delta < 0 && offset > 0 {
			offset = previousRuneStart(d.text, offset)
			delta++
		}
	} else {
		for delta > 0 && offset < len(d.text) {
			offset = nextRuneEnd(d.text, offset)
			delta--
		}
	}
	d.moveCaret(offset, extend)
}

func (d *Document) MoveLine(delta int, extend bool) {
	if d == nil || delta == 0 {
		return
	}
	line, column := d.VisualPosition(d.Caret)
	if !d.desiredValid {
		d.desiredColumn = column
		d.desiredValid = true
	}
	line += delta
	if line < 0 {
		line = 0
	}
	if line >= d.LineCount() {
		line = d.LineCount() - 1
	}
	offset := d.VisualOffset(line, d.desiredColumn)
	if extend {
		d.Caret = offset
	} else {
		d.Anchor = offset
		d.Caret = offset
	}
}

func (d *Document) MoveHome(extend bool) {
	if d == nil {
		return
	}
	line, _ := d.Position(d.Caret)
	d.moveCaret(d.lineStarts[line], extend)
}

func (d *Document) MoveEnd(extend bool) {
	if d == nil {
		return
	}
	line, _ := d.Position(d.Caret)
	d.moveCaret(d.lineEnd(line), extend)
}

func (d *Document) MoveDocumentStart(extend bool) { d.moveCaret(0, extend) }

func (d *Document) MoveDocumentEnd(extend bool) {
	if d != nil {
		d.moveCaret(len(d.text), extend)
	}
}

// Insert adds ordinary typed text and coalesces adjacent typing into one undo
// step. Use Paste for an explicit, independently undoable insertion.
func (d *Document) Insert(text string) {
	d.insertBytes(normalizeLineEndings([]byte(text)), editInsert, true)
}

func (d *Document) Paste(text string) {
	d.insertBytes(normalizeLineEndings([]byte(text)), editReplace, false)
}

func (d *Document) Backspace() bool {
	if d == nil {
		return false
	}
	start, end := d.Selection()
	if start != end {
		d.replace(start, end, nil, editReplace, false)
		return true
	}
	if d.Caret == 0 {
		return false
	}
	start = previousRuneStart(d.text, d.Caret)
	d.replace(start, d.Caret, nil, editBackspace, true)
	return true
}

func (d *Document) Delete() bool {
	if d == nil {
		return false
	}
	start, end := d.Selection()
	if start != end {
		d.replace(start, end, nil, editReplace, false)
		return true
	}
	if d.Caret == len(d.text) {
		return false
	}
	end = nextRuneEnd(d.text, d.Caret)
	d.replace(d.Caret, end, nil, editDelete, true)
	return true
}

func (d *Document) Undo() bool {
	if d == nil || d.historyPosition == 0 {
		return false
	}
	edit := &d.history[d.historyPosition-1]
	d.replaceText(edit.start, edit.start+len(edit.after), edit.before)
	d.Anchor = edit.anchorBefore
	d.Caret = edit.caretBefore
	d.desiredValid = false
	d.historyPosition--
	return true
}

func (d *Document) Redo() bool {
	if d == nil || d.historyPosition >= len(d.history) {
		return false
	}
	edit := &d.history[d.historyPosition]
	d.replaceText(edit.start, edit.start+len(edit.before), edit.after)
	d.Anchor = edit.anchorAfter
	d.Caret = edit.caretAfter
	d.desiredValid = false
	d.historyPosition++
	return true
}

func (d *Document) insertBytes(value []byte, kind editKind, coalesce bool) {
	if d == nil || len(value) == 0 {
		return
	}
	start, end := d.Selection()
	if start != end {
		kind = editReplace
		coalesce = false
	}
	d.replace(start, end, value, kind, coalesce)
}

func (d *Document) replace(start, end int, value []byte, kind editKind, coalesce bool) {
	start = d.clampOffset(start)
	end = d.clampOffset(end)
	if end < start {
		start, end = end, start
	}
	before := appendBytes(nil, d.text[start:end])
	after := appendBytes(nil, value)
	anchorBefore, caretBefore := d.Anchor, d.Caret
	d.replaceText(start, end, after)
	d.Anchor = start + len(after)
	d.Caret = d.Anchor
	d.desiredValid = false

	if d.historyPosition < len(d.history) {
		d.history = d.history[:d.historyPosition]
		if d.savedHistory > d.historyPosition {
			d.savedHistory = -1
		}
	}
	if coalesce && d.savedHistory != d.historyPosition && d.coalesce(kind, start, before, after) {
		return
	}
	d.history = append(d.history, documentEdit{
		start:        start,
		before:       before,
		after:        after,
		anchorBefore: anchorBefore,
		caretBefore:  caretBefore,
		anchorAfter:  d.Anchor,
		caretAfter:   d.Caret,
		kind:         kind,
	})
	d.historyPosition++
}

func (d *Document) coalesce(kind editKind, start int, before, after []byte) bool {
	if d.historyPosition == 0 || d.historyPosition != len(d.history) {
		return false
	}
	last := &d.history[d.historyPosition-1]
	if last.kind != kind {
		return false
	}
	if kind == editInsert && len(before) == 0 && len(last.before) == 0 && start == last.start+len(last.after) {
		last.after = appendBytes(last.after, after)
		last.anchorAfter = d.Anchor
		last.caretAfter = d.Caret
		return true
	}
	if kind == editBackspace && len(after) == 0 && len(last.after) == 0 && start+len(before) == last.start {
		combined := make([]byte, 0, len(before)+len(last.before))
		combined = appendBytes(combined, before)
		combined = appendBytes(combined, last.before)
		last.start = start
		last.before = combined
		last.anchorAfter = d.Anchor
		last.caretAfter = d.Caret
		return true
	}
	if kind == editDelete && len(after) == 0 && len(last.after) == 0 && start == last.start {
		last.before = appendBytes(last.before, before)
		last.anchorAfter = d.Anchor
		last.caretAfter = d.Caret
		return true
	}
	return false
}

func (d *Document) replaceText(start, end int, value []byte) {
	out := make([]byte, 0, len(d.text)-(end-start)+len(value))
	out = appendBytes(out, d.text[:start])
	out = appendBytes(out, value)
	out = appendBytes(out, d.text[end:])
	d.text = out
	d.rebuildLines()
}

func (d *Document) moveCaret(offset int, extend bool) {
	if d == nil {
		return
	}
	offset = d.clampOffset(offset)
	if extend {
		d.Caret = offset
	} else {
		d.Anchor = offset
		d.Caret = offset
	}
	d.desiredValid = false
}

func (d *Document) clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(d.text) {
		return len(d.text)
	}
	for offset > 0 && offset < len(d.text) && isUTF8Continuation(d.text[offset]) {
		offset--
	}
	return offset
}

func (d *Document) rebuildLines() {
	d.lineStarts = d.lineStarts[:0]
	d.lineStarts = append(d.lineStarts, 0)
	for i := 0; i < len(d.text); i++ {
		if d.text[i] == '\n' {
			d.lineStarts = append(d.lineStarts, i+1)
		}
	}
}

func (d *Document) lineForOffset(offset int) int {
	low, high := 0, len(d.lineStarts)
	for low+1 < high {
		middle := low + (high-low)/2
		if d.lineStarts[middle] <= offset {
			low = middle
		} else {
			high = middle
		}
	}
	return low
}

func (d *Document) lineEnd(line int) int {
	if line+1 < len(d.lineStarts) {
		return d.lineStarts[line+1] - 1
	}
	return len(d.text)
}

func detectLineEnding(data []byte) LineEnding {
	crlf, lf := 0, 0
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' {
			continue
		}
		if i > 0 && data[i-1] == '\r' {
			crlf++
		} else {
			lf++
		}
	}
	if crlf > lf {
		return LineEndingCRLF
	}
	return LineEndingLF
}

func normalizeLineEndings(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			continue
		}
		out = append(out, data[i])
	}
	return out
}

func previousRuneStart(data []byte, offset int) int {
	if offset <= 0 {
		return 0
	}
	offset--
	for offset > 0 && isUTF8Continuation(data[offset]) {
		offset--
	}
	return offset
}

func nextRuneEnd(data []byte, offset int) int {
	if offset >= len(data) {
		return len(data)
	}
	offset++
	for offset < len(data) && isUTF8Continuation(data[offset]) {
		offset++
	}
	return offset
}

func isUTF8Continuation(value byte) bool { return value&0xc0 == 0x80 }

func runeCount(data []byte) int {
	count := 0
	for offset := 0; offset < len(data); {
		offset = nextRuneEnd(data, offset)
		count++
	}
	return count
}

func offsetForRuneColumn(data []byte, start, end, column int) int {
	offset := start
	for column > 0 && offset < end {
		offset = nextRuneEnd(data, offset)
		column--
	}
	if offset > end {
		return end
	}
	return offset
}

func visualColumn(data []byte, start, end int) int {
	column := 0
	for at := start; at < end; {
		if data[at] == '\t' {
			column += editorTabWidth
		} else {
			column++
		}
		at = nextRuneEnd(data, at)
	}
	return column
}

func offsetForVisualColumn(data []byte, start, end, column int) int {
	at := start
	visual := 0
	for at < end {
		next := nextRuneEnd(data, at)
		width := 1
		if data[at] == '\t' {
			width = editorTabWidth
		}
		nextVisual := visual + width
		if column < nextVisual {
			if column-visual >= (width+1)/2 {
				return next
			}
			return at
		}
		if column == nextVisual {
			return next
		}
		visual = nextVisual
		at = next
	}
	return end
}

func appendBytes(destination, source []byte) []byte {
	for i := 0; i < len(source); i++ {
		destination = append(destination, source[i])
	}
	return destination
}
