//go:build renvo

package driver

import (
	"renvo.dev/internal/build"
	"renvo.dev/internal/check"
	"renvo.dev/internal/link"
	"renvo.dev/internal/load"
	"renvo.dev/internal/lower"
	"renvo.dev/internal/pipeline"
	"renvo.dev/internal/syntax"
)

type Diagnostic struct {
	Phase   string
	Code    string
	Message string
	Path    string
	Start   int
	End     int
	Line    int
	Column  int
}

const renvoCheckMessageBlob = "duplicate declarationinvalid import declarationinvalid method declarationinvalid function bodyinvalid name or scopereturn value count does not match function resultsassignment value is not assignable to its destinationfeature is not supported by RENVOimport is not usedcalled expression is not a functionleft side of assignment is not assignableassignment count does not matchbreak is not inside a loop or switchcontinue is not inside a loopcall argument is not assignable to its parametergo statement requires a function callchannel direction does not permit this operationselect has an invalid communication clause or multiple defaultslocal variable is declared but not usedpackage main has no top-level func main()func main must have no parameters or resultsmethod main does not define the package entry pointcannot slice an unaddressable array valueconstant array index is out of boundsdeferred builtin call discards a resultinvalid number of arguments to builtininvalid operand type for builtinundefined identifierinvalid operation for operand typesreturn value is not assignable to the function resultfunction call argument count does not match parameters"

var renvoCheckMessageOffsets = [...]int{0, 0, 0, 21, 47, 73, 94, 115, 165, 218, 251, 269, 304, 345, 376, 412, 441, 489, 526, 574, 637, 676, 717, 761, 812, 853, 890, 929, 967, 999, 1019, 1054, 1107, 1161}

const renvoSourceMessageBlob = "go.mod was not foundinvalid module declarationpackage path is outside the main modulepackage directory could not be readsource file could not be readinvalid build constraintsource syntax is invalidunresolved import dependency source is unavailable for dependency version is excluded: dependency has an invalid or missing go.mod: dependency import is ambiguous: invalid go:embed directive or pattern: cgo is not supported by RENVOstandard library package named source files must all be in one directoryexplicit source list contains no buildable Go or C filesC include could not be read: "

var renvoSourceMessageOffsets = [...]int{0, 0, 20, 46, 85, 120, 149, 173, 197, 215, 252, 284, 329, 361, 400, 429, 454, 501, 557, 586}

func renvoCheckMessage(detail int) string {
	return renvoCheckMessageBlob[renvoCheckMessageOffsets[detail]:renvoCheckMessageOffsets[detail+1]]
}

func renvoDiagnosticCode(group string, number int) string {
	text := diagnosticIntText(number)
	if number < 10 {
		text = "00" + text
	} else if number < 100 {
		text = "0" + text
	}
	return "RENVO-" + group + "-" + text
}

const renvoOptionMessageBlob = "missing output after -omissing target after -tunsupported targetmissing tags after -tagsinvalid build tagsmissing package pathextra package path-windows-gui requires a Windows targetmissing arena size after -arena-sizeinvalid arena sizemissing output mode after -mode=unsupported output modekernel-module mode requires linux/amd64invalid renvo:module-license directiveconflicting renvo:module-license directives-script requires one explicit source file-script accepts exactly one source file-emit-unit and -emit-image cannot be used togethermissing system profile after -system-system cannot be combined with -t-system cannot be combined with -arena-sizemissing backend definition after -backendcould not read backend definitioninvalid backend definitionbackend definition does not export target-script requires a .go source fileobject mode requires linux/amd64object mode requires exactly one explicit source file"

var renvoOptionMessageOffsets = [...]int{0, 0, 23, 46, 64, 64, 88, 106, 126, 144, 182, 182, 218, 236, 268, 291, 330, 368, 411, 452, 491, 541, 577, 577, 577, 611, 654, 695, 728, 754, 795, 829, 861, 914, 914}

const renvoSyntaxMessageBlob = "source contains an invalid or unterminated tokeninvalid or missing package clauseinvalid import declarationinvalid top-level declarationinvalid function or method declarationunexpected statement or expression at package scope"

var renvoSyntaxMessageOffsets = [...]int{0, 0, 48, 81, 107, 136, 174, 225}

const renvoLowerMessageBlob = "invalid checked graph reached the lowererinvalid package index reached the lowererinvalid source token reached the lowererpackage unit construction failedunchecked program reached the lowerer"

var renvoLowerMessageOffsets = [...]int{0, 0, 41, 82, 122, 154, 191}

const renvoLinkMessageBlob = "linker received a failed package buildroot package is missinglinked unit is invalid"

var renvoLinkMessageOffsets = [...]int{0, 0, 38, 61, 83}

func renvoOptionMessage(detail int) string {
	if detail > ParseOK && detail+1 < len(renvoOptionMessageOffsets) {
		return renvoOptionMessageBlob[renvoOptionMessageOffsets[detail]:renvoOptionMessageOffsets[detail+1]]
	}
	return ""
}

func renvoSetDiagnostic(d *Diagnostic, phase string, code string, message string) {
	d.Phase, d.Code, d.Message = phase, code, message
}

func renvoSetDiagnosticDetail(d *Diagnostic, code string, message string) {
	d.Code, d.Message = code, message
}

func renvoSyntaxErrorDiagnostic(d *Diagnostic, detail int) {
	if detail > syntax.ParseOK && detail <= syntax.ParseErrTopLevel {
		number := detail + 1
		if detail == syntax.ParseErrScan {
			number = 1
		}
		renvoSetDiagnostic(d, "parser", renvoDiagnosticCode("PARSE", number), renvoSyntaxMessageBlob[renvoSyntaxMessageOffsets[detail]:renvoSyntaxMessageOffsets[detail+1]])
	} else {
		renvoSetDiagnostic(d, "compiler", "RENVO-BUG-004", "compiler bug: parser returned undeclared error code "+diagnosticIntText(detail))
	}
}

func (d Diagnostic) Valid() bool { return d.Code != "" }

func printRenvoDiagnostic(d Diagnostic) {
	if d.Path == "" {
		print("renvo")
	} else {
		print(d.Path)
		if d.Line > 0 {
			print(":")
			renvoPrintInt(d.Line)
			print(":")
			renvoPrintInt(d.Column)
		}
	}
	print(": error ")
	print(d.Code)
	print(" (")
	print(d.Phase)
	print("): ")
	print(d.Message)
	print("\n")
}

func diagnosticForBuild(result BuildResult) Diagnostic {
	d := Diagnostic{Phase: "compiler", Code: "RENVO-BUG-001", Message: "compiler bug: build returned undeclared error code " + diagnosticIntText(result.Error)}
	if result.Error == BuildErrForeign && result.ForeignDiagnostic.Valid() {
		return result.ForeignDiagnostic
	}
	if result.Error == BuildErrOptions {
		renvoSetDiagnostic(&d, "options", "RENVO-OPTION-001", "invalid command options")
		optionError := result.Options.Error
		if optionError == ParseErrUnknownOption {
			renvoSetDiagnosticDetail(&d, "RENVO-OPTION-005", "unknown option "+result.Options.ErrorArg)
		} else if optionError == ParseErrMixedFileList {
			renvoSetDiagnosticDetail(&d, "RENVO-OPTION-011", "explicit source list contains a non-.go/.c argument "+result.Options.ErrorArg)
		} else if optionError == ParseErrSystemRead {
			renvoSetDiagnosticDetail(&d, "RENVO-OPTION-020", result.Options.SystemError)
		} else if optionError == ParseErrInvalidSystem {
			renvoSetDiagnosticDetail(&d, "RENVO-OPTION-021", "invalid system profile "+result.Options.ErrorArg+": "+result.Options.SystemError)
		} else if optionError == ParseErrMissingIncludePath {
			renvoSetDiagnosticDetail(&d, "RENVO-OPTION-032", "missing include directory after "+result.Options.ErrorArg)
		} else {
			message := renvoOptionMessage(optionError)
			if message != "" {
				if optionError == ParseErrUnsupportedTarget || optionError == ParseErrInvalidTags || optionError == ParseErrExtraPackage || optionError == ParseErrInvalidArenaSize || optionError == ParseErrUnsupportedMode || optionError == ParseErrBackendRead || optionError == ParseErrBackendTarget {
					message += " " + result.Options.ErrorArg
				} else if optionError == ParseErrInvalidBackend {
					message += ": " + result.Options.ErrorArg
				}
				number := optionError + 1
				if optionError == ParseErrScriptRequiresFile {
					number = 33
				} else if optionError == ParseErrScriptFileCount {
					number = 34
				} else if optionError == ParseErrConflictingEmit {
					number = 35
				} else if optionError >= ParseErrMissingSystem {
					number = optionError - 2
				}
				renvoSetDiagnosticDetail(&d, renvoDiagnosticCode("OPTION", number), message)
			} else {
				renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-017", "compiler bug: option parser returned undeclared error code "+diagnosticIntText(optionError))
			}
		}
		return d
	}
	if result.Error == BuildErrSource {
		renvoSetDiagnostic(&d, "loader", "RENVO-LOAD-001", "source collection failed")
		d.Path = result.ErrorPath
		sourceError := result.Sources.Error
		if sourceError > SourceOK && sourceError <= SourceErrCInclude {
			d.Message = renvoSourceMessageBlob[renvoSourceMessageOffsets[sourceError]:renvoSourceMessageOffsets[sourceError+1]]
			number := sourceError + 1
			if sourceError == SourceErrParse {
				renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-015", "compiler bug: parser failure source file was not retained")
			} else if sourceError == SourceErrCInclude {
				d.Phase, d.Code = "preprocessor", "RENVO-CPP-001"
			} else {
				if sourceError == SourceErrImport {
					number = 8
				} else if sourceError >= SourceErrDependencyMissing {
					number += 4
				}
				d.Code = renvoDiagnosticCode("LOAD", number)
			}
			if sourceError >= SourceErrImport && sourceError <= SourceErrEmbed || sourceError == SourceErrStandardPackage || sourceError == SourceErrCInclude {
				d.Message += result.ErrorPath
			}
			if sourceError == SourceErrStandardPackage {
				d.Message += " is not included in this RENVO build"
			}
		} else if sourceError == SourceErrCPreprocess {
			return cPreprocessDiagnostic(result.Sources.CPreprocessError, result.Sources.ErrorPath,
				result.Sources.CPreprocessLine, result.Sources.CPreprocessDetail)
		} else if sourceError == SourceErrNoSelectedFiles {
			if result.Sources.ErrorPath == "renvo.dev/device/board" {
				renvoSetDiagnostic(&d, "target", "RENVO-BOARD-001", "device/board requires a supported board target; select a board such as m5nanoc6/riscv32")
			} else {
				renvoSetDiagnostic(&d, "loader", "RENVO-LOAD-024", "package has no files selected by this target and its build tags: "+result.Sources.ErrorPath)
			}
		} else {
			renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-018", "compiler bug: source collector returned undeclared error code "+diagnosticIntText(sourceError))
		}
		if result.Sources.ErrorSourcePath != "" {
			d.Path = result.Sources.ErrorSourcePath
			for i := 0; i < len(result.Sources.Files); i++ {
				if result.Sources.Files[i].Path == d.Path {
					if sourceError == SourceErrParse {
						parsed := syntax.ParseFile(result.Sources.Files[i].Src)
						renvoSyntaxErrorDiagnostic(&d, parsed.Error)
						if parsed.ErrorTok >= 0 && parsed.ErrorTok < len(parsed.Tokens) {
							return renvoDiagnosticOffset(d, result.Sources.Files[i].Src, syntax.TokenStart(parsed.Tokens[parsed.ErrorTok]))
						}
					}
					d = renvoDiagnosticOffset(d, result.Sources.Files[i].Src, result.Sources.ErrorOffset)
				}
			}
		}
		return d
	}
	if result.Error != BuildErrPipeline {
		return d
	}
	built := result.Pipeline
	if built.Error == pipeline.PipelineErrLoad {
		renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-016", "compiler bug: workspace graph failure has no declared graph error")
		workspaceError := built.Workspace.Error
		if workspaceError == load.WorkspaceErrDuplicateFile {
			renvoSetDiagnosticDetail(&d, "RENVO-LOAD-010", "duplicate source file")
			if built.Workspace.ErrorFile >= 0 && built.Workspace.ErrorFile < len(built.Workspace.Files) {
				d.Path = built.Workspace.Files[built.Workspace.ErrorFile].Path
			}
			return d
		} else if workspaceError == load.WorkspaceErrMissingModule {
			renvoSetDiagnosticDetail(&d, "RENVO-LOAD-002", "go.mod was not found")
			return d
		} else if workspaceError == load.WorkspaceErrModule {
			renvoSetDiagnosticDetail(&d, "RENVO-LOAD-003", "invalid module declaration")
			if built.Workspace.ErrorFile >= 0 && built.Workspace.ErrorFile < len(built.Workspace.Files) {
				d.Path = built.Workspace.Files[built.Workspace.ErrorFile].Path
			}
			return d
		} else if workspaceError != load.WorkspaceErrGraph {
			renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-008", "compiler bug: workspace loader returned undeclared error code "+diagnosticIntText(workspaceError))
			return d
		}
		d.Phase = "loader"
		graph := built.Workspace.Graph
		if graph.Error == load.GraphErrCycle {
			renvoSetDiagnosticDetail(&d, "RENVO-LOAD-011", "import cycle detected")
			d.Path = graph.ErrorPath
			for i := 0; i < len(result.Sources.Files); i++ {
				if result.Sources.Files[i].Path == d.Path {
					d = renvoDiagnosticOffset(d, result.Sources.Files[i].Src, graph.ErrorOffset)
				}
			}
			return d
		} else {
			if graph.Error == load.GraphErrRoot {
				renvoSetDiagnosticDetail(&d, "RENVO-LOAD-004", "root package could not be resolved")
			} else if graph.Error != load.GraphErrPackage {
				renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-009", "compiler bug: package graph returned undeclared error code "+diagnosticIntText(graph.Error))
				return d
			}
			pkg := graph.ErrorPackage
			if pkg < 0 {
				pkg = built.Workspace.ErrorFile
			}
			if pkg >= 0 && pkg < len(graph.Packages) {
				packageError := graph.Packages[pkg].Error
				if packageError == load.PackageErrRef {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-023", "package reference could not be resolved")
				} else if packageError == load.PackageErrParse {
					renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-005", "compiler bug: parser failure has no source file coordinate")
				} else if packageError == load.PackageErrC11 {
					d = c11ErrorDiagnostic(d, graph.Packages[pkg].C11Error)
				} else if packageError == load.PackageErrName {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-012", "files in one directory declare different packages")
				} else if packageError == load.PackageErrImport {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-008", "import could not be resolved")
				} else if packageError == load.PackageErrNoFiles {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-013", "package contains no selected Go or C files")
				} else {
					renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-010", "compiler bug: package loader returned undeclared error code "+diagnosticIntText(packageError))
				}
				result.ErrorPackage = pkg
				result.ErrorFile = graph.Packages[pkg].ErrorFile
				if packageError == load.PackageErrParse && result.ErrorFile >= 0 && result.ErrorFile < len(graph.Packages[pkg].Files) {
					file := graph.Packages[pkg].Files[result.ErrorFile]
					renvoSyntaxErrorDiagnostic(&d, file.File.Error)
					if offset := sourceGenericsOffset(file.Src); offset >= 0 {
						renvoSetDiagnosticDetail(&d, "RENVO-PARSE-002", "generics are not supported by RENVO")
						for i := 0; i < len(file.File.Tokens); i++ {
							if syntax.TokenStart(file.File.Tokens[i]) == offset {
								file.File.ErrorTok = i
								break
							}
						}
					}
					result.ErrorToken = file.File.ErrorTok
				}
			}
		}
	} else if built.Error == pipeline.PipelineErrBuild {
		renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-012", "compiler bug: build stage returned undeclared error code "+diagnosticIntText(built.Build.Error))
		if built.Build.Error == build.BuildErrCheck {
			if built.Build.ErrorDetail == check.CheckErrGraph {
				renvoSetDiagnostic(&d, "checker", "RENVO-CHECK-001", "invalid package graph reached the type checker")
			} else if built.Build.ErrorDetail == check.CheckErrTypeAssertion {
				renvoSetDiagnostic(&d, "checker", "RENVO-CHECK-033", "type assertion requires a type; found a composite literal")
			} else if built.Build.ErrorDetail >= check.CheckErrDuplicate && built.Build.ErrorDetail <= check.CheckErrCallArity {
				renvoSetDiagnostic(&d, "checker", renvoDiagnosticCode("CHECK", built.Build.ErrorDetail), renvoCheckMessage(built.Build.ErrorDetail))
			} else {
				renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-013", "compiler bug: type checker returned undeclared error code "+diagnosticIntText(built.Build.ErrorDetail))
			}
		} else if built.Build.Error == build.BuildErrLower {
			detail := built.Build.ErrorDetail
			renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-014", "compiler bug: lowerer returned undeclared error code "+diagnosticIntText(detail))
			if detail > lower.EmitOK && detail < lower.EmitErrAssembly {
				renvoSetDiagnostic(&d, "lowerer", renvoDiagnosticCode("LOWER", detail), renvoLowerMessageBlob[renvoLowerMessageOffsets[detail]:renvoLowerMessageOffsets[detail+1]])
			} else if detail == lower.EmitErrAssembly {
				renvoSetDiagnostic(&d, "rtgasm", "RENVO-RTGASM-001", "invalid RTGASM source or function binding")
				d.Path = built.ErrorPath
				for i := 0; i < len(result.Sources.Files); i++ {
					if result.Sources.Files[i].Path == d.Path {
						return renvoDiagnosticOffset(d, result.Sources.Files[i].Src, built.ErrorOffset)
					}
				}
			}
		} else if built.Build.Error == build.BuildErrUnit {
			renvoSetDiagnostic(&d, "unit", "RENVO-UNIT-001", "lowered package unit is invalid")
		} else if built.Build.Error == build.BuildErrRoot {
			renvoSetDiagnostic(&d, "linker", "RENVO-LINK-002", "root package is missing")
		}
	} else if built.Error == pipeline.PipelineErrLink {
		detail := built.Link.Error
		if detail > link.LinkOK && detail <= link.LinkErrUnit {
			renvoSetDiagnostic(&d, "linker", renvoDiagnosticCode("LINK", detail), renvoLinkMessageBlob[renvoLinkMessageOffsets[detail]:renvoLinkMessageOffsets[detail+1]])
		} else {
			renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-011", "compiler bug: linker returned undeclared error code "+diagnosticIntText(detail))
		}
	} else {
		renvoSetDiagnostic(&d, "compiler", "RENVO-BUG-002", "compiler bug: pipeline returned undeclared error code "+diagnosticIntText(built.Error))
	}
	return renvoBuildDiagnosticLocation(result, d)
}

func renvoBuildDiagnosticLocation(result BuildResult, d Diagnostic) Diagnostic {
	graph := result.Pipeline.Workspace.Graph
	pkg := result.ErrorPackage
	file := result.ErrorFile
	if pkg < 0 || pkg >= len(graph.Packages) {
		return d
	}
	if file < 0 || file >= len(graph.Packages[pkg].Files) {
		d.Path = graph.Packages[pkg].Ref.Dir
		return d
	}
	source := graph.Packages[pkg].Files[file]
	d.Path = source.Path
	if graph.Packages[pkg].Error == load.PackageErrC11 {
		return renvoDiagnosticOffset(d, source.Src, graph.Packages[pkg].ErrorOffset)
	}
	tok := result.ErrorToken
	if tok < 0 || tok >= len(source.File.Tokens) {
		return d
	}
	d.Start = syntax.TokenStart(source.File.Tokens[tok])
	d.End = syntax.TokenEnd(source.File.Tokens[tok])
	if renvoDiagnosticNamesToken(d.Code) && d.Start >= 0 && d.End > d.Start && d.End <= len(source.Src) {
		d.Message += ": " + cDiagnosticIdentifier(source.Path, source.Src[d.Start:d.End])
	}
	d.Line = syntax.TokenLine(source.File.Tokens[tok])
	d.Column = 1
	for i := d.Start - 1; i >= 0 && source.Src[i] != '\n'; i-- {
		d.Column++
	}
	return d
}

func renvoDiagnosticOffset(d Diagnostic, source []byte, offset int) Diagnostic {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	d.Start, d.End = offset, offset
	d.Line, d.Column = 1, 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			d.Line, d.Column = d.Line+1, 1
		} else {
			d.Column++
		}
	}
	return d
}

func renvoDiagnosticNamesToken(code string) bool {
	if len(code) != 15 || code[6] != 'C' {
		return false
	}
	number := int(code[13]-'0')*10 + int(code[14]-'0')
	return number == 10 || number == 11 || number == 20 || number >= 27 && number <= 29 || number == 32
}
