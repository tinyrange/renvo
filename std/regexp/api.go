package regexp

import "unicode/utf8"

// MatchString reports whether s contains any match of the pattern.
func MatchString(pattern string, s string) bool {
	re, err := Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

// Match reports whether b contains any match of the pattern.
func Match(pattern string, b []byte) bool { return MatchString(pattern, string(b)) }

// QuoteMeta escapes all regular expression metacharacters in s so the result
// matches the literal text of s.
func QuoteMeta(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isMeta(c) {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return string(out)
}

func isMeta(c byte) bool {
	switch c {
	case '\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}', '^', '$':
		return true
	}
	return false
}

// MatchString reports whether s contains a match.
func (re *Regexp) MatchString(s string) bool {
	_, ok := re.runFrom(s, 0)
	return ok
}

// Match reports whether b contains a match.
func (re *Regexp) Match(b []byte) bool { return re.MatchString(string(b)) }

// FindString returns the leftmost match, or "" when there is none.
func (re *Regexp) FindString(s string) string {
	loc := re.FindStringIndex(s)
	if loc == nil {
		return ""
	}
	return s[loc[0]:loc[1]]
}

// FindStringIndex returns the byte offsets of the leftmost match as a
// two-element slice, or nil when there is no match.
func (re *Regexp) FindStringIndex(s string) []int {
	caps, ok := re.runFrom(s, 0)
	if !ok {
		return nil
	}
	loc := make([]int, 2)
	loc[0] = caps[0]
	loc[1] = caps[1]
	return loc
}

// FindStringSubmatch returns the leftmost match and its capturing groups. The
// returned slice has 2*(groups+1) entries; unmatched groups are "". It
// returns nil when there is no match.
func (re *Regexp) FindStringSubmatch(s string) []string {
	caps, ok := re.runFrom(s, 0)
	if !ok {
		return nil
	}
	out := make([]string, re.slots/2)
	for i := 0; i+1 < len(caps); i += 2 {
		if caps[i] < 0 || caps[i] > caps[i+1] || caps[i+1] > len(s) {
			out[i/2] = ""
			continue
		}
		out[i/2] = s[caps[i]:caps[i+1]]
	}
	return out
}

// FindAllString returns successive non-overlapping matches. A negative n
// returns all matches; otherwise at most n are returned. An empty match
// adjacent to the previous match is skipped by advancing one rune.
func (re *Regexp) FindAllString(s string, n int) []string {
	var out []string
	pos := 0
	for n < 0 || len(out) < n {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		start, end := caps[0], caps[1]
		out = append(out, s[start:end])
		if end == start {
			_, width := utf8.DecodeRuneInString(s[end:])
			if width == 0 {
				width = 1
			}
			pos = end + width
			if pos > len(s) {
				break
			}
			continue
		}
		pos = end
	}
	return out
}

// Count reports the number of non-overlapping matches in s.
func (re *Regexp) Count(s string) int { return len(re.FindAllString(s, -1)) }

// ReplaceAllString replaces every match with repl. The replacement may refer
// to capturing groups with $1 through $9; $$ inserts a literal dollar sign.
func (re *Regexp) ReplaceAllString(s string, repl string) string {
	out := make([]byte, 0, len(s))
	pos := 0
	for pos <= len(s) {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		start, end := caps[0], caps[1]
		out = append(out, s[pos:start]...)
		out = expand(re, repl, s, caps, out)
		if end == start {
			_, width := utf8.DecodeRuneInString(s[end:])
			if width == 0 {
				width = 1
			}
			if end+width > len(s) {
				out = append(out, s[end:]...)
				return string(out)
			}
			out = append(out, s[end:end+width]...)
			pos = end + width
			continue
		}
		pos = end
	}
	if pos < len(s) {
		out = append(out, s[pos:]...)
	}
	return string(out)
}

func expand(re *Regexp, repl string, s string, caps []int, out []byte) []byte {
	for i := 0; i < len(repl); i++ {
		if repl[i] != '$' || i+1 >= len(repl) {
			out = append(out, repl[i])
			continue
		}
		i++
		if repl[i] == '$' {
			out = append(out, '$')
			continue
		}
		if repl[i] < '1' || repl[i] > '9' {
			out = append(out, '$', repl[i])
			continue
		}
		group := int(repl[i]-'0') * 2
		if group+1 < len(caps) && caps[group] >= 0 && caps[group] <= caps[group+1] && caps[group+1] <= len(s) {
			out = append(out, s[caps[group]:caps[group+1]]...)
		}
	}
	return out
}

// advanceAfterMatch returns the position where matching resumes after the
// match described by caps. Non-empty matches resume at their end; empty
// matches advance one rune so they cannot repeat forever.
func advanceAfterMatch(s string, caps []int) int {
	start, end := caps[0], caps[1]
	if end > start {
		return end
	}
	_, width := utf8.DecodeRuneInString(s[end:])
	if width == 0 {
		width = 1
	}
	return end + width
}

// FindAllStringIndex returns the byte offsets of successive non-overlapping
// matches as two-element slices. A negative n returns all matches; otherwise
// at most n are returned. It returns nil when there is no match.
func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	var out [][]int
	pos := 0
	for n < 0 || len(out) < n {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		out = append(out, []int{caps[0], caps[1]})
		pos = advanceAfterMatch(s, caps)
		if pos > len(s) {
			break
		}
	}
	return out
}

// FindStringSubmatchIndex returns the byte offsets of the leftmost match and
// its capturing groups as a slice of 2*(groups+1) entries; unmatched groups
// are recorded as -1, -1 pairs. It returns nil when there is no match.
func (re *Regexp) FindStringSubmatchIndex(s string) []int {
	caps, ok := re.runFrom(s, 0)
	if !ok {
		return nil
	}
	out := make([]int, len(caps))
	copy(out, caps)
	return out
}

// FindAllStringSubmatch returns the submatches of successive non-overlapping
// matches. A negative n returns all matches; otherwise at most n.
func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	var out [][]string
	pos := 0
	for n < 0 || len(out) < n {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		sub := make([]string, re.slots/2)
		for i := 0; i+1 < len(caps); i += 2 {
			if caps[i] < 0 || caps[i] > caps[i+1] || caps[i+1] > len(s) {
				sub[i/2] = ""
				continue
			}
			sub[i/2] = s[caps[i]:caps[i+1]]
		}
		out = append(out, sub)
		pos = advanceAfterMatch(s, caps)
		if pos > len(s) {
			break
		}
	}
	return out
}

// Split slices s into substrings separated by matches of the expression,
// returning a slice with at most n substrings when n > 0. Text containing no
// match is returned as a single element.
func (re *Regexp) Split(s string, n int) []string {
	if n == 0 {
		return nil
	}
	if len(s) == 0 {
		return []string{s}
	}
	matches := re.FindAllStringIndex(s, n-1)
	out := make([]string, 0, len(matches)+1)
	beg := 0
	end := 0
	for _, match := range matches {
		if n > 0 && len(out) >= n-1 {
			break
		}
		end = match[0]
		if match[1] != 0 {
			out = append(out, s[beg:end])
		}
		beg = match[1]
	}
	if end != len(s) {
		out = append(out, s[beg:])
	}
	return out
}

// ReplaceAllLiteralString replaces every match with repl without expanding
// dollar references in the replacement text.
func (re *Regexp) ReplaceAllLiteralString(s string, repl string) string {
	out := make([]byte, 0, len(s))
	pos := 0
	for pos <= len(s) {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		start, end := caps[0], caps[1]
		out = append(out, s[pos:start]...)
		out = append(out, repl...)
		next := advanceAfterMatch(s, caps)
		if next <= len(s) {
			out = append(out, s[end:next]...)
		} else {
			out = append(out, s[end:]...)
			return string(out)
		}
		pos = next
	}
	if pos < len(s) {
		out = append(out, s[pos:]...)
	}
	return string(out)
}

// ReplaceAllStringFunc replaces every match with the result of calling repl
// on the matched text.
func (re *Regexp) ReplaceAllStringFunc(s string, repl func(string) string) string {
	out := make([]byte, 0, len(s))
	pos := 0
	for pos <= len(s) {
		caps, ok := re.runFrom(s, pos)
		if !ok {
			break
		}
		start, end := caps[0], caps[1]
		out = append(out, s[pos:start]...)
		out = append(out, repl(s[start:end])...)
		next := advanceAfterMatch(s, caps)
		if next <= len(s) {
			out = append(out, s[end:next]...)
		} else {
			out = append(out, s[end:]...)
			return string(out)
		}
		pos = next
	}
	if pos < len(s) {
		out = append(out, s[pos:]...)
	}
	return string(out)
}
