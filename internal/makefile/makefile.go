// Package makefile parses and evaluates the deliberately small Makefile
// language used by "renvo make". Recipes invoke Renvo commands directly;
// they are not passed through a platform shell, so the same file behaves in
// the native command and the browser virtual filesystem.
package makefile

type Variable struct {
	Name  string
	Value string
}

type Rule struct {
	Targets       []string
	Prerequisites []string
	Recipes       []string
	Line          int
	Phony         bool
}

type File struct {
	Variables []Variable
	Rules     []Rule
	Default   string
}

type Error struct {
	Line    int
	Message string
}

type Command struct {
	Args   []string
	Text   string
	Quiet  bool
	Target string
	Line   int
}

// StampFunc reports a file's modification stamp. The actual time unit is not
// important; stamps are only compared with other values from the same call.
type StampFunc func(path string) (int64, bool)

func Parse(src []byte) (File, Error) {
	file := File{}
	lines := splitLines(src)
	var current *Rule
	phony := make([]string, 0, 4)
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		line := trimRight(lines[lineIndex])
		lineNumber := lineIndex + 1
		if line == "" || firstNonSpace(line) == '#' {
			continue
		}
		if line[0] == '\t' {
			if current == nil {
				return File{}, Error{Line: lineNumber, Message: "recipe has no preceding rule"}
			}
			recipe := trimSpace(line[1:])
			if recipe != "" {
				current.Recipes = append(current.Recipes, recipe)
			}
			continue
		}
		current = nil
		if name, operator, value, ok := assignment(line); ok {
			value = expand(value, file.Variables, nil)
			setVariable(&file, name, operator, value)
			continue
		}
		colon := findUnquoted(line, ':')
		if colon < 0 {
			return File{}, Error{Line: lineNumber, Message: "expected a variable assignment or target rule"}
		}
		targets, wordError := words(expand(trimSpace(line[:colon]), file.Variables, nil))
		if wordError != "" || len(targets) == 0 {
			return File{}, Error{Line: lineNumber, Message: "target list is malformed"}
		}
		dependencies, wordError := words(expand(trimComment(line[colon+1:]), file.Variables, nil))
		if wordError != "" {
			return File{}, Error{Line: lineNumber, Message: wordError}
		}
		if len(targets) == 1 && targets[0] == ".PHONY" {
			phony = append(phony, dependencies...)
			continue
		}
		file.Rules = append(file.Rules, Rule{Targets: targets, Prerequisites: dependencies, Line: lineNumber})
		current = &file.Rules[len(file.Rules)-1]
		if file.Default == "" && len(targets[0]) > 0 && targets[0][0] != '.' {
			file.Default = targets[0]
		}
	}
	for i := 0; i < len(file.Rules); i++ {
		for j := 0; j < len(file.Rules[i].Targets); j++ {
			if contains(phony, file.Rules[i].Targets[j]) {
				file.Rules[i].Phony = true
			}
		}
	}
	if file.Default == "" && len(file.Rules) > 0 {
		file.Default = file.Rules[0].Targets[0]
	}
	return file, Error{}
}

// Plan returns commands in dependency order. When exists is nil, all rules
// are evaluated; this is used by browser builds whose virtual filesystem does
// not expose stable modification times. Native callers can skip leaf targets
// which already exist by supplying exists.
func Plan(file File, targets []string, stamp StampFunc) ([]Command, Error) {
	if len(targets) == 0 {
		if file.Default == "" {
			return nil, Error{Message: "Makefile contains no targets"}
		}
		targets = []string{file.Default}
	}
	commands := make([]Command, 0, 8)
	visiting := make([]string, 0, 8)
	done := make([]string, 0, 8)
	dirty := make([]string, 0, 8)
	var visit func(string) (bool, Error)
	visit = func(target string) (bool, Error) {
		if contains(done, target) {
			return contains(dirty, target), Error{}
		}
		if contains(visiting, target) {
			return false, Error{Message: "dependency cycle contains " + target}
		}
		rule := findRule(file.Rules, target)
		if rule == nil {
			if stamp != nil {
				_, present := stamp(target)
				if present {
					done = append(done, target)
					return false, Error{}
				}
			}
			return false, Error{Message: "no rule to make target " + target}
		}
		visiting = append(visiting, target)
		dependencyDirty := false
		for i := 0; i < len(rule.Prerequisites); i++ {
			dependency := rule.Prerequisites[i]
			if findRule(file.Rules, dependency) != nil {
				rebuilt, err := visit(dependency)
				if err.Message != "" {
					return false, err
				}
				dependencyDirty = dependencyDirty || rebuilt
			} else if stamp != nil {
				if _, present := stamp(dependency); !present {
					return false, Error{Line: rule.Line, Message: "no rule to make prerequisite " + dependency + " for " + target}
				}
			}
		}
		visiting = visiting[:len(visiting)-1]
		shouldRun := rule.Phony || stamp == nil || dependencyDirty
		if !shouldRun {
			targetStamp, present := stamp(target)
			shouldRun = !present
			if present {
				for i := 0; i < len(rule.Prerequisites); i++ {
					dependencyStamp, dependencyPresent := stamp(rule.Prerequisites[i])
					if !dependencyPresent || dependencyStamp > targetStamp {
						shouldRun = true
						break
					}
				}
			}
		}
		if shouldRun {
			automatic := []Variable{{Name: "@", Value: target}}
			if len(rule.Prerequisites) > 0 {
				automatic = append(automatic, Variable{Name: "<", Value: rule.Prerequisites[0]})
				automatic = append(automatic, Variable{Name: "^", Value: join(rule.Prerequisites, " ")})
			}
			for i := 0; i < len(rule.Recipes); i++ {
				text := expand(rule.Recipes[i], file.Variables, automatic)
				quiet := len(text) > 0 && text[0] == '@'
				if quiet {
					text = trimSpace(text[1:])
				}
				args, wordError := words(text)
				if wordError != "" {
					return false, Error{Line: rule.Line + i + 1, Message: wordError}
				}
				if len(args) == 0 {
					continue
				}
				if args[0] != "renvo" {
					return false, Error{Line: rule.Line + i + 1, Message: "recipes must invoke renvo directly"}
				}
				commands = append(commands, Command{Args: args, Text: text, Quiet: quiet, Target: target, Line: rule.Line + i + 1})
			}
		}
		done = append(done, target)
		if shouldRun {
			dirty = append(dirty, target)
		}
		return shouldRun, Error{}
	}
	for i := 0; i < len(targets); i++ {
		if _, err := visit(targets[i]); err.Message != "" {
			return nil, err
		}
	}
	return commands, Error{}
}

func findRule(rules []Rule, target string) *Rule {
	for i := 0; i < len(rules); i++ {
		if contains(rules[i].Targets, target) {
			return &rules[i]
		}
	}
	return nil
}

func assignment(line string) (string, string, string, bool) {
	operators := []string{":=", "?=", "+=", "="}
	for i := 0; i < len(operators); i++ {
		at := index(line, operators[i])
		if at <= 0 {
			continue
		}
		name := trimSpace(line[:at])
		if name == "" || indexByte(name, ':') >= 0 || indexByte(name, ' ') >= 0 || indexByte(name, '\t') >= 0 {
			continue
		}
		return name, operators[i], trimSpace(trimComment(line[at+len(operators[i]):])), true
	}
	return "", "", "", false
}

func setVariable(file *File, name string, operator string, value string) {
	for i := 0; i < len(file.Variables); i++ {
		if file.Variables[i].Name != name {
			continue
		}
		if operator == "?=" {
			return
		}
		if operator == "+=" && file.Variables[i].Value != "" {
			file.Variables[i].Value += " " + value
		} else {
			file.Variables[i].Value = value
		}
		return
	}
	file.Variables = append(file.Variables, Variable{Name: name, Value: value})
}

func expand(text string, variables []Variable, automatic []Variable) string {
	out := make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if text[i] != '$' || i+1 >= len(text) {
			out = append(out, text[i])
			continue
		}
		i++
		if text[i] == '$' {
			out = append(out, '$')
			continue
		}
		name := ""
		if text[i] == '(' || text[i] == '{' {
			close := byte(')')
			if text[i] == '{' {
				close = '}'
			}
			start := i + 1
			i = start
			for i < len(text) && text[i] != close {
				i++
			}
			if i >= len(text) {
				out = append(out, text[start-2:]...)
				break
			}
			name = text[start:i]
		} else {
			name = text[i : i+1]
		}
		value := variableValue(automatic, name)
		if value == "" {
			value = variableValue(variables, name)
		}
		out = append(out, value...)
	}
	return string(out)
}

func words(text string) ([]string, string) {
	out := make([]string, 0, 8)
	word := make([]byte, 0, 32)
	quote := byte(0)
	escaped := false
	flush := func() {
		if len(word) > 0 {
			out = append(out, string(word))
			word = word[:0]
		}
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if escaped {
			word = append(word, ch)
			escaped = false
			continue
		}
		if ch == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				word = append(word, ch)
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
		} else if ch == ' ' || ch == '\t' {
			flush()
		} else {
			word = append(word, ch)
		}
	}
	if escaped || quote != 0 {
		return nil, "unterminated quote or escape in recipe"
	}
	flush()
	return out, ""
}

func splitLines(src []byte) []string {
	out := make([]string, 0, 16)
	start := 0
	for i := 0; i <= len(src); i++ {
		if i == len(src) || src[i] == '\n' {
			line := string(src[start:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	return out
}

func trimComment(text string) string {
	quote := byte(0)
	for i := 0; i < len(text); i++ {
		if quote != 0 {
			if text[i] == quote {
				quote = 0
			}
		} else if text[i] == '\'' || text[i] == '"' {
			quote = text[i]
		} else if text[i] == '#' {
			return trimSpace(text[:i])
		}
	}
	return trimSpace(text)
}

func trimSpace(text string) string {
	start, end := 0, len(text)
	for start < end && (text[start] == ' ' || text[start] == '\t') {
		start++
	}
	for end > start && (text[end-1] == ' ' || text[end-1] == '\t') {
		end--
	}
	return text[start:end]
}
func trimRight(text string) string {
	for len(text) > 0 && (text[len(text)-1] == ' ' || text[len(text)-1] == '\t') {
		text = text[:len(text)-1]
	}
	return text
}
func firstNonSpace(text string) byte {
	for i := 0; i < len(text); i++ {
		if text[i] != ' ' && text[i] != '\t' {
			return text[i]
		}
	}
	return 0
}
func contains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
func variableValue(values []Variable, name string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i].Name == name {
			return values[i].Value
		}
	}
	return ""
}
func join(values []string, separator string) string {
	out := ""
	for i := 0; i < len(values); i++ {
		if i > 0 {
			out += separator
		}
		out += values[i]
	}
	return out
}
func indexByte(text string, ch byte) int {
	for i := 0; i < len(text); i++ {
		if text[i] == ch {
			return i
		}
	}
	return -1
}
func index(text string, part string) int {
	if part == "" {
		return 0
	}
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
func findUnquoted(text string, wanted byte) int {
	quote := byte(0)
	for i := 0; i < len(text); i++ {
		if quote != 0 {
			if text[i] == quote {
				quote = 0
			}
		} else if text[i] == '\'' || text[i] == '"' {
			quote = text[i]
		} else if text[i] == wanted {
			return i
		}
	}
	return -1
}
