package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/unit"
)

// LinkBuildCoreIncremental links independently reusable package artifacts.
// Package-local import removal, symbol redirection, and namespacing are cached;
// the final pass only lays out those artifacts and performs transformations
// whose semantics genuinely span package boundaries.
func LinkBuildCoreIncremental(result build.Result) Result {
	InitializePackageArtifactCache()
	session := BeginPackageSession(result, false)
	for !session.Step() {
	}
	return session.Result()
}

// PackageSession resolves or reuses at most one independently linked package
// artifact per Step call. The final Step assembles the compact backend unit.
type PackageSession struct {
	build           build.Result
	transient       bool
	stage           int
	packageNext     int
	prepared        []unit.Program
	symbolOffsets   []int
	aliases         []string
	plusReplacement int
	contextA        int
	contextB        int
	artifacts       []unit.Program
	artifactStarts  []int
	artifactEnds    []int
	result          Result
}

func BeginPackageSession(input build.Result, transient bool) *PackageSession {
	InitializePackageArtifactCache()
	return &PackageSession{
		build:     input,
		transient: transient,
		result:    Result{Ok: true, Error: LinkOK, ErrorPackage: -1},
	}
}

func (s *PackageSession) Step() bool {
	if s == nil || s.stage >= 3 {
		return true
	}
	if s.stage == 0 {
		if !s.build.Ok {
			s.result.Ok = false
			s.result.Error = LinkErrBuild
			s.result.ErrorPackage = s.build.ErrorPackage
			s.stage = 3
			return true
		}
		if s.build.Root < 0 || s.build.Root >= len(s.build.Units) {
			s.result.Ok = false
			s.result.Error = LinkErrRoot
			s.stage = 3
			return true
		}
		programs := make([]unit.Program, len(s.build.Units))
		for i := 0; i < len(s.build.Units); i++ {
			programs[i] = s.build.Units[i].Program
		}
		var ok bool
		s.prepared, ok = prepareProgramsCore(programs, s.build.Root)
		if !ok {
			s.failUnit()
			return true
		}
		ensureCoreProgramSymbols(s.prepared)
		s.symbolOffsets = corePackageSymbolOffsets(s.prepared)
		s.aliases = corePackageSymbolAliases(s.prepared, s.build.Root, s.symbolOffsets)
		s.contextA, s.contextB = incrementalArtifactContextHash(s.prepared, s.aliases, s.build.Root)
		s.plusReplacement = len(s.aliases)
		s.aliases = append(s.aliases, "+")
		s.artifacts = make([]unit.Program, len(s.prepared))
		s.artifactStarts = make([]int, len(s.prepared))
		s.artifactEnds = make([]int, len(s.prepared))
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		if s.packageNext < len(s.prepared) {
			i := s.packageNext
			s.packageNext++
			s.artifactStarts[i] = arena.Mark()
			artifact, hit := loadPackageArtifact(s.build.Units[i], i, s.contextA, s.contextB)
			if !hit {
				var ok bool
				artifact, ok = linkOnePackageArtifactCore(s.prepared[i], s.aliases, s.symbolOffsets, s.plusReplacement)
				if !ok {
					s.failUnit()
					return true
				}
				storePackageArtifact(s.build.Units[i], i, s.contextA, s.contextB, artifact)
			}
			s.artifacts[i] = artifact
			s.artifactEnds[i] = arena.Mark()
			if s.transient {
				arena.Discard(s.build.Units[i].ArenaStart, s.build.Units[i].ArenaEnd)
			}
			return false
		}
		s.stage = 2
		return false
	}
	program, ok := mergePackageArtifactsCore(s.artifacts, s.build.Units, s.build.Root, s.build.Units[s.build.Root].Name)
	if !ok {
		s.failUnit()
		return true
	}
	for i := 0; i < len(s.artifacts); i++ {
		arena.Discard(s.artifactStarts[i], s.artifactEnds[i])
	}
	if !lowerFunctionValuesCore(&program, s.transient) {
		s.failUnit()
		return true
	}
	compactCoreLinkedTokenLines(program.Tokens)
	var data []byte
	if s.transient {
		data, ok = unit.MarshalCoreTransient(unit.CoreProgramFrom(program))
	} else {
		data, ok = unit.MarshalCore(unit.CoreProgramFrom(program))
	}
	if !ok {
		s.failUnit()
		return true
	}
	if !s.transient {
		s.result.Program = program
	}
	s.result.Data = data
	s.stage = 3
	return true
}

func (s *PackageSession) Result() Result {
	if s == nil {
		return Result{Ok: false, Error: LinkErrUnit, ErrorPackage: -1}
	}
	return s.result
}

func (s *PackageSession) failUnit() {
	s.result.Ok = false
	s.result.Error = LinkErrUnit
	s.stage = 3
}

func linkOnePackageArtifactCore(src unit.Program, aliases []string, symbolOffsets []int, plusReplacement int) (unit.Program, bool) {
	var empty unit.Program
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return empty, false
	}
	// Keep actions separate from token source lines. The linker uses those lines
	// directly when laying out the artifact, avoiding another scan of its text.
	actions := make([]int, len(src.Tokens))
	if !linkedTokenActions(&src, &aliases, symbolOffsets, actions, plusReplacement) {
		return empty, false
	}
	finalEOF := 0
	for i := 0; i < len(actions); i++ {
		if src.Tokens[i].KindLine&255 != unit.TokenEOF && !tokenActionSkips(actions[i]) {
			finalEOF++
			if incrementalTokenIsEllipsis(src.Tokens[i], src.Text) {
				finalEOF += 2
			}
		}
	}
	artifact := unit.Program{Package: cloneCoreLinkString(src.Package), ImportPath: cloneCoreLinkString(src.ImportPath)}
	artifact.Text = make([]byte, 0, len(src.Text))
	artifact.Tokens = make([]unit.Token, 0, finalEOF+1)
	artifact.Decls = make([]unit.Decl, 0, len(src.Decls))
	artifact.Funcs = make([]unit.Func, 0, len(src.Funcs))
	ok, line := appendProgramCore(&artifact, src, actions, finalEOF, 1, aliases, false)
	if !ok {
		return empty, false
	}
	artifact.Tokens = append(artifact.Tokens, unit.MakeToken(unit.TokenEOF, len(artifact.Text), 0, line))
	return artifact, true
}

func incrementalTokenIsEllipsis(token unit.Token, text []byte) bool {
	return token.KindLine&255 == unit.TokenOp && token.Size == 3 && token.Start >= 0 && token.Start+2 < len(text) && text[token.Start] == '.' && text[token.Start+1] == '.' && text[token.Start+2] == '.'
}

func mergePackageArtifactsCore(artifacts []unit.Program, units []build.PackageUnit, root int, rootName string) (unit.Program, bool) {
	var empty unit.Program
	if root < 0 || root >= len(artifacts) || len(units) != len(artifacts) || rootName == "" {
		return empty, false
	}
	finalEOF := 0
	textCapacity := 0
	declCapacity := 0
	funcCapacity := 0
	for i := 0; i < len(artifacts); i++ {
		if len(artifacts[i].Tokens) == 0 || artifacts[i].Tokens[len(artifacts[i].Tokens)-1].KindLine&255 != unit.TokenEOF {
			return empty, false
		}
		finalEOF += len(artifacts[i].Tokens) - 1
		textCapacity += len(artifacts[i].Text) + 1
		declCapacity += len(artifacts[i].Decls)
		funcCapacity += len(artifacts[i].Funcs)
	}
	program := unit.Program{Package: cloneCoreLinkString(rootName), ImportPath: cloneCoreLinkString(artifacts[root].ImportPath)}
	program.Text = make([]byte, 0, textCapacity)
	program.Tokens = make([]unit.Token, 0, finalEOF+1)
	program.Decls = make([]unit.Decl, 0, declCapacity)
	program.Funcs = make([]unit.Func, 0, funcCapacity)
	line := 1
	for i := 0; i < len(artifacts); i++ {
		info := beginLinkedPackageInfo(&program, units[i])
		var ok bool
		line, ok = appendPackageArtifactCore(&program, artifacts[i], line, i+1 < len(artifacts))
		if !ok {
			return empty, false
		}
		finishLinkedPackageInfo(&program, info)
	}
	program.Tokens = append(program.Tokens, unit.MakeToken(unit.TokenEOF, len(program.Text), 0, line))
	return program, true
}

func appendPackageArtifactCore(dst *unit.Program, src unit.Program, line int, hasNext bool) (int, bool) {
	if len(src.Tokens) == 0 || src.Tokens[len(src.Tokens)-1].KindLine&255 != unit.TokenEOF {
		return line, false
	}
	textBase := len(dst.Text)
	tokenBase := len(dst.Tokens)
	localEOF := len(src.Tokens) - 1
	dst.Text = appendCoreBytes(dst.Text, src.Text)
	for i := 0; i < localEOF; i++ {
		tok := src.Tokens[i]
		tok.Start += textBase
		tok.KindLine = tok.KindLine&255 | ((tok.KindLine>>8)+line-1)<<8
		dst.Tokens = append(dst.Tokens, tok)
	}
	mapToken := func(token int) int {
		if token == localEOF {
			return tokenBase + localEOF
		}
		if token < 0 || token > localEOF {
			return -1
		}
		return tokenBase + token
	}
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.NameStart += textBase
		decl.NameEnd += textBase
		decl.StartTok = mapToken(decl.StartTok)
		decl.EndTok = mapToken(decl.EndTok)
		if decl.StartTok < 0 || decl.EndTok < 0 {
			return line, false
		}
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.NameStart += textBase
		fn.NameEnd += textBase
		fn.StartTok = mapToken(fn.StartTok)
		fn.NameTok = mapToken(fn.NameTok)
		if fn.ReceiverStart != 0 || fn.ReceiverEnd != 0 {
			fn.ReceiverStart = mapToken(fn.ReceiverStart)
			fn.ReceiverEnd = mapToken(fn.ReceiverEnd)
		}
		fn.BodyStart = mapToken(fn.BodyStart)
		fn.BodyEnd = mapToken(fn.BodyEnd)
		fn.EndTok = mapToken(fn.EndTok)
		if fn.StartTok < 0 || fn.NameTok < 0 || fn.ReceiverStart < 0 || fn.ReceiverEnd < 0 || fn.BodyStart < 0 || fn.BodyEnd < 0 || fn.EndTok < 0 {
			return line, false
		}
		dst.Funcs = append(dst.Funcs, fn)
	}
	line += countCoreNewlines(src.Text)
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
		line++
	}
	return line, true
}

func incrementalArtifactContextHash(programs []unit.Program, aliases []string, root int) (int, int) {
	a, b := 101, 211
	a, b = incrementalArtifactHashInt(a, b, root)
	for i := 0; i < len(programs); i++ {
		a, b = incrementalArtifactHashString(a, b, programs[i].ImportPath)
		for j := 0; j < len(programs[i].Symbols); j++ {
			a, b = incrementalArtifactHashString(a, b, programs[i].Symbols[j].Name)
		}
		for j := 0; j < len(programs[i].Funcs); j++ {
			name := coreLinkedProgramText(programs[i], programs[i].Funcs[j].NameStart, programs[i].Funcs[j].NameEnd)
			if name == "init" || name == "renvo_runtime_SetProcess" {
				a, b = incrementalArtifactHashString(a, b, name)
			}
		}
	}
	for i := 0; i < len(aliases); i++ {
		a, b = incrementalArtifactHashString(a, b, aliases[i])
	}
	return a, b
}

func incrementalArtifactHashString(a int, b int, value string) (int, int) {
	for i := 0; i < len(value); i++ {
		a, b = incrementalArtifactHashInt(a, b, int(value[i]))
	}
	return incrementalArtifactHashInt(a, b, len(value))
}

func incrementalArtifactHashInt(a int, b int, value int) (int, int) {
	return a*131 + value + 1, b*257 + value + 3
}
