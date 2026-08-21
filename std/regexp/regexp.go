package regexp

import (
	"strconv"
	"unicode/utf8"
)

// Regexp is a compiled regular expression. A Regexp value is safe for
// concurrent use by multiple tasks.
type Regexp struct {
	expr  string
	prog  []*inst
	ncap  int
	slots int
	start int
}

// Compile parses expr and returns a compiled Regexp. The accepted syntax is a
// practical subset of Go's RE2 syntax: literals, ., character classes with
// ranges and negation, the Perl classes \d \D \w \W \s \S, anchors ^ and $,
// capturing and non-capturing groups, alternation, and greedy or lazy
// quantifiers * + ? {n} {n,} {n,m}. Matching runs in O(len(input) times
// len(program)) regardless of the pattern, so user-supplied patterns cannot
// trigger catastrophic backtracking.
func Compile(expr string) (*Regexp, error) {
	ast, ncap, err := parse(expr)
	if err != nil {
		return nil, err
	}
	re := &Regexp{expr: expr, ncap: ncap, slots: 2*ncap + 2}
	c := &compiler{prog: []*inst{}}
	save0 := c.add(&inst{op: opSave, x: 0})
	c.emit(ast)
	c.prog[save0].y = c.fstart
	save1 := c.add(&inst{op: opSave, x: 1})
	c.patchEdgesOn(c.fout, save1)
	matchPC := c.add(&inst{op: opMatch})
	c.prog[save1].y = matchPC
	re.prog = c.prog
	re.start = save0
	return re, nil
}

// MustCompile is like Compile but panics on failure.
func MustCompile(expr string) *Regexp {
	re, err := Compile(expr)
	if err != nil {
		panic("regexp: Compile(" + strconv.Quote(expr) + "): " + err.Error())
	}
	return re
}

// String returns the source pattern.
func (re *Regexp) String() string { return re.expr }

// NumSubexp returns the number of capturing groups.
func (re *Regexp) NumSubexp() int { return re.ncap }

// The emitter keeps the current fragment in compiler fields instead of
// passing fragment structs around: Renvo currently miscompiles some
// struct-with-slice copies (see COMPILER_BUGS.md), so this code sticks to
// plain ints, local slices, and slices returned from calls.
//
// pending edges: v >= 0 patches prog[v].x, v < 0 patches prog[-v-1].y

// makeEdges returns a one-element edge list built inside a call, which is
// the assignment shape the backend compiles correctly today.
func makeEdges(v int) []int {
	return []int{v}
}

// patchEdge writes target into the pending edge encoded by v.
func patchEdge(prog []*inst, v int, target int) {
	if v < 0 {
		prog[-v-1].y = target
		return
	}
	prog[v].x = target
}

// patchEdgesOn patches pending edges against the compiler's program.
func (c *compiler) patchEdgesOn(out []int, target int) {
	for _, v := range out {
		patchEdge(c.prog, v, target)
	}
}

type compiler struct {
	prog   []*inst
	fstart int
	fout   []int
}

func (c *compiler) add(i *inst) int {
	c.prog = append(c.prog, i)
	return len(c.prog) - 1
}

func (c *compiler) setFrag(start int, out []int) {
	c.fstart = start
	c.fout = out
}

// emit compiles n into the program, leaving the resulting fragment in
// c.fstart / c.fout.
func (c *compiler) emit(n *node) {
	switch n.kind {
	case nodeEmpty:
		pc := len(c.prog)
		c.setFrag(pc, makeEdges(pc))
		return
	case nodeClass:
		pc := c.add(&inst{op: opClass, cls: n.cls})
		c.setFrag(pc, makeEdges(pc))
		return
	case nodeAny:
		pc := c.add(&inst{op: opAny})
		c.setFrag(pc, makeEdges(pc))
		return
	case nodeBegin:
		pc := c.add(&inst{op: opBegin})
		c.setFrag(pc, makeEdges(pc))
		return
	case nodeEnd:
		pc := c.add(&inst{op: opEnd})
		c.setFrag(pc, makeEdges(pc))
		return
	case nodeGroup:
		save1 := c.add(&inst{op: opSave, x: n.cap})
		c.emit(n.child)
		innerStart := c.fstart
		innerOut := c.fout
		c.prog[save1].y = innerStart
		save2 := c.add(&inst{op: opSave, x: n.cap + 1})
		c.patchEdgesOn(innerOut, save2)
		c.setFrag(save1, makeEdges(-(save2 + 1)))
		return
	case nodeConcat:
		c.emit(n.kids[0])
		first := c.fstart
		currentOut := c.fout
		for i := 1; i < len(n.kids); i++ {
			c.emit(n.kids[i])
			nextStart := c.fstart
			nextOut := c.fout
			c.patchEdgesOn(currentOut, nextStart)
			currentOut = nextOut
		}
		c.setFrag(first, currentOut)
		return
	case nodeAlternate:
		splits := []int{}
		ends := []int{}
		for i := 0; i < len(n.kids); i++ {
			if i == len(n.kids)-1 {
				c.emit(n.kids[i])
				ends = append(ends, c.fout...)
				for _, s := range splits {
					c.prog[s].y = c.fstart
				}
				break
			}
			split := c.add(&inst{op: opSplit})
			splits = append(splits, split)
			c.emit(n.kids[i])
			c.prog[split].x = c.fstart
			ends = append(ends, c.fout...)
		}
		c.setFrag(splits[0], ends)
		return
	case nodeRepeat:
		c.emitRepeat(n)
		return
	}
	pc := len(c.prog)
	c.setFrag(pc, makeEdges(pc))
}

func (c *compiler) emitRepeat(n *node) {
	currentStart := -1
	currentOut := []int{}
	for i := 0; i < n.min; i++ {
		c.emit(n.child)
		if currentStart < 0 {
			currentStart = c.fstart
			currentOut = c.fout
		} else {
			c.patchEdgesOn(currentOut, c.fstart)
			currentOut = c.fout
		}
	}
	if n.max == -1 {
		c.emit(n.child)
		bodyStart := c.fstart
		bodyOut := c.fout
		jmp := c.add(&inst{op: opJmp})
		c.patchEdgesOn(bodyOut, jmp)
		loop := c.add(&inst{op: opSplit, y: bodyStart})
		c.prog[jmp].x = loop
		after := len(c.prog)
		if n.lazy {
			// Prefer exiting; retry the body only when the exit fails.
			c.prog[loop].x = after
		} else {
			c.prog[loop].x = bodyStart
			c.prog[loop].y = after
		}
		if currentStart < 0 {
			c.setFrag(loop, nil)
			return
		}
		c.patchEdgesOn(currentOut, loop)
		c.setFrag(currentStart, nil)
		return
	}
	optional := n.max - n.min
	if optional <= 0 {
		if currentStart < 0 {
			pc := len(c.prog)
			c.setFrag(pc, makeEdges(pc))
			return
		}
		c.setFrag(currentStart, currentOut)
		return
	}
	if !n.lazy {
		exits := []int{}
		chainStart := -1
		chainOut := []int{}
		for i := 0; i < optional; i++ {
			split := c.add(&inst{op: opSplit})
			c.emit(n.child)
			bodyStart := c.fstart
			bodyOut := c.fout
			c.prog[split].x = bodyStart
			exits = append(exits, -(split + 1))
			if chainStart < 0 {
				chainStart = split
			} else {
				c.patchEdgesOn(chainOut, split)
			}
			chainOut = bodyOut
		}
		out := append(chainOut, exits...)
		if currentStart < 0 {
			c.setFrag(chainStart, out)
			return
		}
		c.patchEdgesOn(currentOut, chainStart)
		c.setFrag(currentStart, out)
		return
	}
	// Lazy bounded repeats prefer skipping each optional copy, so the chain is
	// built from the last copy backwards: each split tries the exit first and
	// falls back to its body.
	start := -1
	outs := []int{}
	for i := 0; i < optional; i++ {
		c.emit(n.child)
		bodyStart := c.fstart
		bodyOut := c.fout
		split := c.add(&inst{op: opSplit, y: bodyStart})
		if start < 0 {
			// The final unit exits through the split's preferred (x) arm.
			outs = append(outs, split)
			outs = append(outs, bodyOut...)
		} else {
			c.prog[split].x = start
			c.patchEdgesOn(bodyOut, start)
		}
		start = split
	}
	if currentStart < 0 {
		c.setFrag(start, outs)
		return
	}
	c.patchEdgesOn(currentOut, start)
	c.setFrag(currentStart, outs)
}

// thread is one NFA state plus its capture slots.
type thread struct {
	pc   int
	caps []int
}

// vm carries the state of one execution over the program. When a match is
// recorded, cut is the length of the list at that moment: later (lower
// priority) threads are pruned, while earlier ones continue and may extend
// or improve the match.
type vm struct {
	re      *Regexp
	s       string
	visited []int
	gen     int
	matched bool
	cut     int
	best    []int
}

// run executes the program against s starting at index from. Every start
// position at or after from is tried in turn; the result has leftmost-first
// semantics identical to Go's regexp package.
func (re *Regexp) runFrom(s string, from int) ([]int, bool) {
	v := &vm{re: re, s: s, visited: make([]int, len(re.prog))}
	base := freshSlots(re.slots)
	var clist []*thread
	var nlist []*thread
	gen := 0
	for pos := from; pos <= len(s); pos++ {
		gen++
		v.gen = gen
		if !v.matched {
			clist = v.add(clist, re.start, pos, base)
		}
		gen++
		v.gen = gen
		nlist = nlist[:0]
		for i := 0; i < len(clist); i++ {
			if v.matched && i >= v.cut {
				break
			}
			t := clist[i]
			in := re.prog[t.pc]
			switch in.op {
			case opClass:
				if pos < len(s) {
					r, size := utf8.DecodeRuneInString(s[pos:])
					if in.cls.matches(r) {
						nlist = v.add(nlist, in.x, pos+size, t.caps)
					}
				}
			case opAny:
				if pos < len(s) {
					r, size := utf8.DecodeRuneInString(s[pos:])
					if r != '\n' {
						nlist = v.add(nlist, in.x, pos+size, t.caps)
					}
				}
			case opMatch:
				v.matched = true
				v.cut = i
				v.best = t.caps
			}
		}
		if v.matched && len(nlist) == 0 {
			break
		}
		clist, nlist = nlist, clist[:0]
	}
	return v.best, v.matched
}

// add follows splits, jumps, saves, and anchors, appending runnable threads
// to list. Caps may be mutated and restored during Save traversal; threads
// receive private copies when appended.
func (v *vm) add(list []*thread, pc int, pos int, caps []int) []*thread {
	if pc < 0 || pc >= len(v.re.prog) {
		return list
	}
	if v.visited[pc] == v.gen {
		return list
	}
	v.visited[pc] = v.gen
	in := v.re.prog[pc]
	switch in.op {
	case opJmp:
		return v.add(list, in.x, pos, caps)
	case opSplit:
		list = v.add(list, in.x, pos, caps)
		return v.add(list, in.y, pos, caps)
	case opSave:
		if in.x < len(caps) {
			saved := caps[in.x]
			caps[in.x] = pos
			list = v.add(list, in.y, pos, caps)
			caps[in.x] = saved
			return list
		}
		return v.add(list, in.y, pos, caps)
	case opBegin:
		if pos != 0 {
			return list
		}
		return v.add(list, pc+1, pos, caps)
	case opEnd:
		if pos != len(v.s) {
			return list
		}
		return v.add(list, pc+1, pos, caps)
	case opMatch:
		v.matched = true
		v.cut = len(list)
		v.best = copySlots(caps)
		return list
	}
	return append(list, &thread{pc: pc, caps: copySlots(caps)})
}

func copySlots(caps []int) []int {
	out := make([]int, len(caps))
	copy(out, caps)
	return out
}

func freshSlots(count int) []int {
	out := make([]int, count)
	for i := range out {
		out[i] = -1
	}
	return out
}
