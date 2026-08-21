package regexp

import "unicode/utf8"

// Error codes reported by Compile for unsupported or malformed patterns.
type ErrorCode int

const (
	ErrInternalError ErrorCode = iota
	ErrInvalidCharClass
	ErrInvalidCharRange
	ErrInvalidEscape
	ErrInvalidRepeatOp
	ErrInvalidRepeatSize
	ErrMissingBracket
	ErrMissingParen
	ErrTrailingBackslash
	ErrUnexpectedParen
)

// Error describes a pattern compilation failure.
type Error struct {
	Code ErrorCode
	Expr string
}

func (e *Error) Error() string {
	names := []string{
		"internal error",
		"invalid character class",
		"invalid character class range",
		"invalid escape sequence",
		"invalid repeat operator",
		"invalid repeat count",
		"missing closing ]",
		"missing closing )",
		"trailing backslash at end of expression",
		"unexpected )",
	}
	index := int(e.Code)
	if index < 0 || index >= len(names) {
		index = 0
	}
	return "error parsing regexp: " + names[index] + ": `" + e.Expr + "`"
}

// AST node kinds.
const (
	nodeClass = iota
	nodeAny
	nodeBegin
	nodeEnd
	nodeEmpty
	nodeConcat
	nodeAlternate
	nodeRepeat
	nodeGroup
)

// charClass matches one rune against inclusive ranges. negated inverts the
// result for classes such as [^a-z].
type charClass struct {
	ranges  []rune // pairs of lo, hi
	negated bool
}

func (cc *charClass) matches(r rune) bool {
	in := false
	for i := 0; i+1 < len(cc.ranges); i += 2 {
		if r >= cc.ranges[i] && r <= cc.ranges[i+1] {
			in = true
			break
		}
	}
	if cc.negated {
		return !in
	}
	return in
}

type node struct {
	kind  int
	cls   *charClass
	kids  []*node
	child *node
	min   int
	max   int // -1 means unbounded
	lazy  bool
	cap   int // capture slot base for nodeGroup
}

// Program opcodes.
const (
	opClass = iota
	opAny
	opSplit
	opJmp
	opSave
	opMatch
	opBegin
	opEnd
)

type inst struct {
	op  int
	cls *charClass
	x   int
	y   int
}

const maxRepeatCount = 1000

type parser struct {
	expr string
	pos  int
	ncap int
}

func parse(expr string) (*node, int, error) {
	p := &parser{expr: expr}
	alt, err := p.parseAlternate()
	if err != nil {
		return nil, 0, err
	}
	if p.pos < len(p.expr) {
		if p.expr[p.pos] == ')' {
			return nil, 0, &Error{Code: ErrUnexpectedParen, Expr: expr}
		}
		return nil, 0, &Error{Code: ErrInternalError, Expr: expr}
	}
	return alt, p.ncap, nil
}

func (p *parser) peek() byte {
	if p.pos < len(p.expr) {
		return p.expr[p.pos]
	}
	return 0
}

func (p *parser) parseAlternate() (*node, error) {
	options := []*node{}
	for {
		item, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		options = append(options, item)
		if p.peek() == '|' {
			p.pos++
			continue
		}
		break
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return &node{kind: nodeAlternate, kids: options}, nil
}

func (p *parser) parseConcat() (*node, error) {
	items := []*node{}
	for len(p.expr) > p.pos && p.peek() != '|' && p.peek() != ')' {
		item, err := p.parseRepeat()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return &node{kind: nodeEmpty}, nil
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &node{kind: nodeConcat, kids: items}, nil
}

func (p *parser) parseRepeat() (*node, error) {
	atom, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	min, max, ok, err := p.parseQuantifier()
	if err != nil {
		return nil, err
	}
	if !ok {
		return atom, nil
	}
	switch atom.kind {
	case nodeBegin, nodeEnd:
		return nil, &Error{Code: ErrInvalidRepeatOp, Expr: p.expr}
	case nodeRepeat:
		return nil, &Error{Code: ErrInvalidRepeatOp, Expr: p.expr}
	}
	if min > maxRepeatCount || (max >= 0 && max > maxRepeatCount) {
		return nil, &Error{Code: ErrInvalidRepeatSize, Expr: p.expr}
	}
	if max >= 0 && max < min {
		return nil, &Error{Code: ErrInvalidRepeatSize, Expr: p.expr}
	}
	if p.peek() == '?' {
		p.pos++
		return &node{kind: nodeRepeat, child: atom, min: min, max: max, lazy: true}, nil
	}
	return &node{kind: nodeRepeat, child: atom, min: min, max: max}, nil
}

func (p *parser) parseQuantifier() (int, int, bool, error) {
	c := p.peek()
	if c == '*' {
		p.pos++
		return 0, -1, true, nil
	}
	if c == '+' {
		p.pos++
		return 1, -1, true, nil
	}
	if c == '?' {
		p.pos++
		return 0, 1, true, nil
	}
	if c != '{' {
		return 0, 0, false, nil
	}
	save := p.pos
	p.pos++
	min, ok := p.parseNumber()
	if !ok {
		p.pos = save
		return 0, 0, false, nil
	}
	max := min
	if p.peek() == ',' {
		p.pos++
		if p.peek() == '}' {
			max = -1
		} else {
			value, ok := p.parseNumber()
			if !ok {
				return 0, 0, false, &Error{Code: ErrInvalidRepeatSize, Expr: p.expr}
			}
			max = value
		}
	}
	if p.peek() != '}' {
		p.pos = save
		return 0, 0, false, nil
	}
	p.pos++
	return min, max, true, nil
}

func (p *parser) parseNumber() (int, bool) {
	start := p.pos
	value := 0
	for p.pos < len(p.expr) && p.expr[p.pos] >= '0' && p.expr[p.pos] <= '9' {
		value = value*10 + int(p.expr[p.pos]-'0')
		if value > maxRepeatCount*10 {
			value = maxRepeatCount*10 + 1
		}
		p.pos++
	}
	return value, p.pos > start
}

func (p *parser) parseAtom() (*node, error) {
	c := p.peek()
	switch c {
	case '(':
		p.pos++
		if p.peek() == '?' {
			p.pos++
			if p.peek() == ':' {
				p.pos++
				inner, err := p.parseAlternate()
				if err != nil {
					return nil, err
				}
				if p.peek() != ')' {
					return nil, &Error{Code: ErrMissingParen, Expr: p.expr}
				}
				p.pos++
				return inner, nil
			}
			return nil, &Error{Code: ErrInvalidEscape, Expr: p.expr}
		}
		p.ncap++
		base := 2 * p.ncap
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, &Error{Code: ErrMissingParen, Expr: p.expr}
		}
		p.pos++
		return &node{kind: nodeGroup, cap: base, child: inner}, nil
	case ')':
		return nil, &Error{Code: ErrUnexpectedParen, Expr: p.expr}
	case '[':
		return p.parseClass()
	case '.':
		p.pos++
		return &node{kind: nodeAny}, nil
	case '^':
		p.pos++
		return &node{kind: nodeBegin}, nil
	case '$':
		p.pos++
		return &node{kind: nodeEnd}, nil
	case '*':
		return nil, &Error{Code: ErrInvalidRepeatOp, Expr: p.expr}
	case '+':
		return nil, &Error{Code: ErrInvalidRepeatOp, Expr: p.expr}
	case '?':
		return nil, &Error{Code: ErrInvalidRepeatOp, Expr: p.expr}
	case '\\':
		return p.parseEscape()
	}
	r, size := utf8.DecodeRuneInString(p.expr[p.pos:])
	p.pos += size
	return &node{kind: nodeClass, cls: singleClass(r)}, nil
}

// singleClass builds a one-range class. The ranges slice is built with make
// and index stores because Renvo miscompiles composite slice literals inside
// returned composite structs (see COMPILER_BUGS.md).
// Class range slices are always built inside small helpers and assigned from
// their results: Renvo miscompiles slice allocations that are stored into a
// struct field by the same function that allocated them (see COMPILER_BUGS.md).

func singleClass(r rune) *charClass {
	cc := &charClass{}
	cc.ranges = runePair(r, r)
	return cc
}

func runePair(lo rune, hi rune) []rune {
	rs := make([]rune, 2)
	rs[0] = lo
	rs[1] = hi
	return rs
}

func digitRanges() []rune {
	return runePair('0', '9')
}

func (p *parser) parseEscape() (*node, error) {
	p.pos++ // consume backslash
	if p.pos >= len(p.expr) {
		return nil, &Error{Code: ErrTrailingBackslash, Expr: p.expr}
	}
	c := p.expr[p.pos]
	p.pos++
	switch c {
	case 'd':
		dc := &charClass{}
		dc.ranges = digitRanges()
		return &node{kind: nodeClass, cls: dc}, nil
	case 'D':
		dc := &charClass{negated: true}
		dc.ranges = digitRanges()
		return &node{kind: nodeClass, cls: dc}, nil
	case 'w':
		return &node{kind: nodeClass, cls: wordClass(false)}, nil
	case 'W':
		return &node{kind: nodeClass, cls: wordClass(true)}, nil
	case 's':
		return &node{kind: nodeClass, cls: spaceClass(false)}, nil
	case 'S':
		return &node{kind: nodeClass, cls: spaceClass(true)}, nil
	case 't':
		return &node{kind: nodeClass, cls: singleClass('\t')}, nil
	case 'n':
		return &node{kind: nodeClass, cls: singleClass('\n')}, nil
	case 'r':
		return &node{kind: nodeClass, cls: singleClass('\r')}, nil
	case 'f':
		return &node{kind: nodeClass, cls: singleClass('\f')}, nil
	case 'v':
		return &node{kind: nodeClass, cls: singleClass('\v')}, nil
	case 'a':
		return &node{kind: nodeClass, cls: singleClass(7)}, nil
	}
	if c >= '0' && c <= '7' {
		return nil, &Error{Code: ErrInvalidEscape, Expr: p.expr}
	}
	if isWordByte(c) {
		return nil, &Error{Code: ErrInvalidEscape, Expr: p.expr}
	}
	return &node{kind: nodeClass, cls: singleClass(rune(c))}, nil
}

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func wordRanges() []rune {
	rs := make([]rune, 8)
	rs[0] = '0'
	rs[1] = '9'
	rs[2] = 'A'
	rs[3] = 'Z'
	rs[4] = '_'
	rs[5] = '_'
	rs[6] = 'a'
	rs[7] = 'z'
	return rs
}

func spaceRanges() []rune {
	rs := make([]rune, 6)
	rs[0] = '\t'
	rs[1] = '\n'
	rs[2] = 12
	rs[3] = 13
	rs[4] = ' '
	rs[5] = ' '
	return rs
}

func wordClass(negated bool) *charClass {
	cc := &charClass{negated: negated}
	cc.ranges = wordRanges()
	return cc
}

func spaceClass(negated bool) *charClass {
	cc := &charClass{negated: negated}
	cc.ranges = spaceRanges()
	return cc
}

func (p *parser) parseClass() (*node, error) {
	p.pos++ // consume [
	cc := &charClass{}
	if p.peek() == '^' {
		cc.negated = true
		p.pos++
	}
	members := 0
	var rs []rune
	for p.peek() != ']' {
		if p.pos >= len(p.expr) {
			return nil, &Error{Code: ErrMissingBracket, Expr: p.expr}
		}
		lo, err := p.classRune()
		if err != nil {
			return nil, err
		}
		hi := lo
		if p.peek() == '-' && p.pos+1 < len(p.expr) && p.expr[p.pos+1] != ']' {
			p.pos++
			hi, err = p.classRune()
			if err != nil {
				return nil, err
			}
			if hi < lo {
				return nil, &Error{Code: ErrInvalidCharRange, Expr: p.expr}
			}
		}
		rs = append(rs, lo, hi)
		members++
	}
	if members == 0 {
		return nil, &Error{Code: ErrMissingBracket, Expr: p.expr}
	}
	p.pos++ // consume ]
	cc.ranges = rs
	return &node{kind: nodeClass, cls: cc}, nil
}

func (p *parser) classRune() (rune, error) {
	c := p.peek()
	if c == '\\' {
		p.pos++
		if p.pos >= len(p.expr) {
			return 0, &Error{Code: ErrTrailingBackslash, Expr: p.expr}
		}
		e := p.expr[p.pos]
		p.pos++
		switch e {
		case 'd', 'D', 'w', 'W', 's', 'S':
			return 0, &Error{Code: ErrInvalidCharClass, Expr: p.expr}
		case 't':
			return '\t', nil
		case 'n':
			return '\n', nil
		case 'r':
			return '\r', nil
		case 'f':
			return 12, nil
		case 'v':
			return 11, nil
		case 'a':
			return 7, nil
		}
		if isWordByte(e) {
			return 0, &Error{Code: ErrInvalidEscape, Expr: p.expr}
		}
		return rune(e), nil
	}
	r, size := utf8.DecodeRuneInString(p.expr[p.pos:])
	p.pos += size
	return r, nil
}
