package build

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/lower"
)

// ProgramSession checks and lowers at most one package per Step call. It is
// the resumable form of the ordinary build path used by event-loop clients.
type ProgramSession struct {
	graph       load.Graph
	transient   bool
	cached      bool
	stage       int
	packageNext int
	headerStart int
	headerEnd   int
	graphKeyA   int
	graphKeyB   int
	contextA    []int
	contextB    []int
	checked     check.Program
	result      Result
}

func BeginProgramsSession(graph load.Graph, transient bool, cached bool) *ProgramSession {
	if cached {
		InitializePackageProgramCache()
	}
	graphKeyA, graphKeyB := packageGraphHash(graph)
	contextA, contextB := packageContextHashes(graph)
	return &ProgramSession{
		graph:     graph,
		transient: transient,
		cached:    cached,
		graphKeyA: graphKeyA,
		graphKeyB: graphKeyB,
		contextA:  contextA,
		contextB:  contextB,
		result: Result{
			Root:         -1,
			Ok:           true,
			Error:        BuildOK,
			ErrorPackage: -1,
			ErrorFile:    -1,
			ErrorToken:   -1,
		},
	}
}

// Step returns true once the session has completed, successfully or not.
func (s *ProgramSession) Step() bool {
	if s == nil || s.stage >= 2 {
		return true
	}
	if s.stage == 0 {
		s.headerStart = arena.Mark()
		s.checked = check.CheckGraphHeadersCore(s.graph)
		s.headerEnd = arena.Mark()
		if !s.checked.Ok {
			s.result.ErrorDetail = s.checked.Error
			s.result = buildFail(s.result, BuildErrCheck, s.checked.ErrorPackage, s.checked.ErrorFile, s.checked.ErrorToken)
			s.stage = 2
			return true
		}
		s.stage = 1
		return false
	}
	if s.packageNext >= len(s.graph.Packages) {
		if s.result.Root < 0 {
			s.result = buildFail(s.result, BuildErrRoot, -1, -1, -1)
		}
		if s.transient {
			arena.Discard(s.headerStart, s.headerEnd)
		}
		s.stage = 2
		return true
	}
	i := s.packageNext
	s.packageNext++
	pkg := s.graph.Packages[i]
	persistMark := 0
	if s.transient {
		persistMark = arena.PersistMark()
	}
	sourceKeyA, sourceKeyB := packageSourceHash(pkg)
	isRoot := pkg.Ref.ImportPath == s.graph.Root
	contextA := s.graphKeyA
	contextB := s.graphKeyB
	if i < len(s.contextA) && i < len(s.contextB) {
		contextA = s.contextA[i]
		contextB = s.contextB[i]
	}
	s.checked = check.CheckGraphPackageCore(s.graph, s.checked, i)
	if !s.checked.Ok {
		if s.transient {
			arena.PersistReset(persistMark)
		}
		s.result.ErrorDetail = s.checked.Error
		s.result = buildFail(s.result, BuildErrCheck, s.checked.ErrorPackage, s.checked.ErrorFile, s.checked.ErrorToken)
		s.stage = 2
		return true
	}
	if isRoot && pkg.Name == "main" {
		if mainErr, mainFile, mainTok := check.CheckRootMain(pkg); mainErr != check.CheckOK {
			if s.transient {
				arena.PersistReset(persistMark)
			}
			s.result.ErrorDetail = mainErr
			s.result = buildFail(s.result, BuildErrCheck, i, mainFile, mainTok)
			s.stage = 2
			return true
		}
	}
	unitStart := arena.Mark()
	if s.cached && !isRoot {
		cachedProgram, hit := loadCachedPackageProgram(s.graph, i, contextA, contextB)
		if hit {
			unitEnd := arena.Mark()
			s.result.Units = append(s.result.Units, PackageUnit{
				ImportPath: cachedProgram.ImportPath,
				Name:       cachedProgram.Package,
				Program:    cachedProgram,
				GraphKeyA:  contextA,
				GraphKeyB:  contextB,
				SourceKeyA: sourceKeyA,
				SourceKeyB: sourceKeyB,
				ArenaStart: unitStart,
				ArenaEnd:   unitEnd,
			})
			s.discardPackage(pkg)
			if s.transient {
				arena.PersistReset(persistMark)
			}
			return false
		}
	}
	emit := lower.EmitCheckedPackageCore(pkg, s.checked.Packages[i], s.transient)
	unitEnd := arena.Mark()
	if !emit.Ok {
		if s.transient {
			arena.PersistReset(persistMark)
		}
		s.result.ErrorDetail = emit.Error
		s.result = buildFail(s.result, BuildErrLower, i, emit.ErrorFile, emit.ErrorToken)
		s.stage = 2
		return true
	}
	if isRoot {
		s.result.Root = len(s.result.Units)
	}
	if s.cached && !isRoot {
		storeCachedPackageProgram(s.graph, i, contextA, contextB, emit.Program)
	}
	s.result.Units = append(s.result.Units, PackageUnit{
		ImportPath: emit.Program.ImportPath,
		Name:       emit.Program.Package,
		Program:    emit.Program,
		GraphKeyA:  contextA,
		GraphKeyB:  contextB,
		SourceKeyA: sourceKeyA,
		SourceKeyB: sourceKeyB,
		ArenaStart: unitStart,
		ArenaEnd:   unitEnd,
	})
	s.discardPackage(pkg)
	if s.transient {
		arena.PersistReset(persistMark)
	}
	return false
}

func (s *ProgramSession) Result() Result {
	if s == nil {
		return Result{Root: -1, Ok: false, Error: BuildErrRoot, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
	}
	return s.result
}

func (s *ProgramSession) discardPackage(pkg load.Package) {
	if !s.transient {
		return
	}
	for i := 0; i < len(pkg.Files); i++ {
		arena.Discard(pkg.Files[i].ArenaStart, pkg.Files[i].ArenaEnd)
	}
	arena.Discard(pkg.CoreArenaStart, pkg.CoreArenaEnd)
}
