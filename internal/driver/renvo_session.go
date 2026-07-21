//go:build renvo

package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/backendbridge"
	"renvo.dev/internal/build"
	"renvo.dev/internal/link"
	"renvo.dev/std/os"
)

// RenvoCommandSession is the resumable embedded compiler command. Frontend
// phases advance package by package, and supported backends emit bounded
// batches of reusable machine-code objects between event-loop yields.
type RenvoCommandSession struct {
	args          []string
	env           []string
	stage         int
	done          bool
	resetArena    bool
	mark          int
	build         *FSBuildSession
	built         BuildResult
	backend       *backendbridge.CompileSession
	persistMark   int
	virtualTarget string
	backendOutput string
	status        int
	diagnostic    string
	cached        bool
}

func BeginRenvoCommand(args []string, env []string) *RenvoCommandSession {
	return beginRenvoCommand(args, env, true)
}

func beginRenvoCommand(args []string, env []string, cached bool) *RenvoCommandSession {
	return &RenvoCommandSession{args: args, env: env, cached: cached}
}

func (s *RenvoCommandSession) Step() bool {
	if s == nil || s.done {
		return true
	}
	if s.stage == 0 {
		if CommandHelpRequested(s.args) {
			s.diagnostic = HelpText
			s.done = true
			return true
		}
		if len(renvoCommandDiagnosticBuffer) == 0 {
			renvoCommandDiagnosticBuffer = make([]byte, renvoCommandDiagnosticCapacity)
		}
		if s.cached {
			build.InitializePackageProgramCache()
			link.InitializePackageArtifactCache()
		}
		commandArgs := s.args
		if len(commandArgs) > 0 {
			commandArgs = commandArgs[1:]
		}
		objectTarget := DefaultTarget
		for i := 0; i+1 < len(commandArgs); i++ {
			if commandArgs[i] == "-t" {
				objectTarget = commandArgs[i+1]
				break
			}
		}
		if s.cached {
			backendbridge.InitializeObjectCache(objectTarget)
		}
		s.resetArena = renvoFrontendCanResetArena()
		if s.resetArena {
			s.mark = arena.Mark()
		}
		s.build = beginFSBuildSession(commandArgs, renvoWorkDir(s.env), renvoStdRoot(s.args, s.env), renvoModuleCache(s.env), RenvoFS{}, true, s.cached)
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		if !s.build.Step() {
			return false
		}
		s.built = s.build.Result()
		if !s.built.Ok {
			s.fail(s.built.Diagnostic, s.resetArena, s.mark)
			return true
		}
		if s.built.CacheHit {
			if s.resetArena {
				arena.Reset(s.mark)
			}
			s.done = true
			return true
		}
		s.stage = 2
		return false
	}
	if s.stage == 2 {
		if s.built.Options.EmitUnit {
			if s.built.Options.Output == "-" {
				print(string(s.built.Unit))
			} else if os.WriteFile(s.built.Options.Output, s.built.Unit, 0644) != nil {
				s.fail(Diagnostic{Phase: "unit", Code: "RENVO-UNIT-002", Message: "failed to write linked unit"}, s.resetArena, s.mark)
				return true
			}
			if s.resetArena {
				arena.Reset(s.mark)
			}
			s.done = true
			return true
		}
		s.stage = 3
		return false
	}
	if s.backend == nil {
		unit := s.built.Unit
		target := s.built.Options.Target
		output := s.built.Options.Output
		if s.resetArena {
			s.persistMark = arena.PersistMark()
			unit = arena.PersistBytes(unit)
			target = arena.PersistString(target)
			output = arena.PersistString(output)
			backendMark := s.mark
			remainder := backendMark % 4096
			if remainder != 0 {
				backendMark = backendMark + 4096 - remainder
			}
			arena.Reset(backendMark)
		}
		s.virtualTarget = target
		s.backendOutput = output
		target = backendTarget(target)
		arenaSize := backendArenaSize(target, s.built.Options.Tags, s.built.Options.ArenaSize)
		s.backend = backendbridge.BeginCompileSession(unit, target, output, s.built.Options.Strip, s.built.Options.WindowsGUI, arenaSize)
		return false
	}
	if !s.backend.Step() {
		return false
	}
	ok := s.backend.Result()
	if ok && s.virtualTarget == "browser/wasm32" {
		wasm, readErr := os.ReadFile(s.backendOutput)
		if readErr != nil || os.WriteFile(s.backendOutput, PackageBrowserHTML(wasm), 0644) != nil {
			ok = false
		}
	}
	if s.resetArena {
		arena.PersistReset(s.persistMark)
		arena.Reset(s.mark)
	}
	if !ok {
		s.fail(Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-001", Message: "backend compilation failed"}, false, 0)
		return true
	}
	if s.resetArena && s.cached {
		rememberEmbeddedBuild(s.built)
	}
	s.done = true
	return true
}

func (s *RenvoCommandSession) Result() (int, string) {
	if s == nil {
		return 1, "renvo: compiler session is unavailable"
	}
	return s.status, s.diagnostic
}

func (s *RenvoCommandSession) fail(diagnostic Diagnostic, resetArena bool, mark int) {
	s.status, s.diagnostic = finishRenvoCommandFailure(renvoCommandDiagnosticBuffer, diagnostic, resetArena, mark)
	s.done = true
}
