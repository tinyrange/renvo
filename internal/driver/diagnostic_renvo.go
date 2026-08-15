//go:build renvo

package driver

import (
	"renvo.dev/internal/build"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
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

var renvoCheckMessageOffsets = [...]int{0, 0, 0, 21, 47, 73, 94, 115, 165, 218, 251, 269, 304, 345, 376, 412, 441, 489, 530, 577, 636, 675, 716, 760, 811, 852, 889, 928, 966, 998, 1018, 1053, 1106, 1160}

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

func renvoOptionMessage(detail int) string {
	switch detail {
	case ParseErrInvalidModuleLicense:
		return "invalid renvo:module-license directive"
	case ParseErrConflictingModuleLicense:
		return "conflicting renvo:module-license directives"
	case ParseErrMissingSystem:
		return "missing system profile after -system"
	case ParseErrSystemTargetConflict:
		return "-system cannot be combined with -t"
	case ParseErrSystemArenaConflict:
		return "-system cannot be combined with -arena-size"
	case ParseErrScriptRequiresGo:
		return "-script requires a .go source file"
	case ParseErrObjectRequiresLinuxAmd64:
		return "object mode requires linux/amd64"
	case ParseErrObjectFileCount:
		return "object mode requires exactly one explicit source file"
	}
	return ""
}

func renvoSetDiagnostic(d *Diagnostic, phase string, code string, message string) {
	d.Phase, d.Code, d.Message = phase, code, message
}

func renvoSetDiagnosticDetail(d *Diagnostic, code string, message string) {
	d.Code, d.Message = code, message
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
	d := Diagnostic{Phase: "frontend", Code: "RENVO-FRONTEND-001", Message: "frontend build failed"}
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
				number := optionError + 1
				if optionError >= ParseErrMissingSystem {
					number = optionError - 2
				}
				renvoSetDiagnosticDetail(&d, renvoDiagnosticCode("OPTION", number), message)
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
				d.Phase, d.Code = "parser", "RENVO-PARSE-001"
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
		}
		if result.Sources.ErrorSourcePath != "" {
			d.Path = result.Sources.ErrorSourcePath
			for i := 0; i < len(result.Sources.Files); i++ {
				if result.Sources.Files[i].Path == d.Path {
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
		renvoSetDiagnostic(&d, "loader", "RENVO-LOAD-009", "workspace loading failed")
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
			pkg := graph.ErrorPackage
			if pkg < 0 {
				pkg = built.Workspace.ErrorFile
			}
			if pkg >= 0 && pkg < len(graph.Packages) {
				packageError := graph.Packages[pkg].Error
				if packageError == load.PackageErrParse {
					renvoSetDiagnostic(&d, "parser", "RENVO-PARSE-001", "source syntax is invalid")
				} else if packageError == load.PackageErrC11 {
					renvoSetDiagnostic(&d, "c11", "RENVO-C11-001", "C11 source is not supported or is invalid")
					if graph.Packages[pkg].C11Error == c11.TranslateErrVLA {
						renvoSetDiagnosticDetail(&d, "RENVO-C11-002", "variable length arrays are not supported")
					}
				} else if packageError == load.PackageErrName {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-012", "files in one directory declare different packages")
				} else if packageError == load.PackageErrImport {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-008", "import could not be resolved")
				} else if packageError == load.PackageErrNoFiles {
					renvoSetDiagnosticDetail(&d, "RENVO-LOAD-013", "package contains no selected Go or C files")
				}
				result.ErrorPackage = pkg
				result.ErrorFile = graph.Packages[pkg].ErrorFile
				if packageError == load.PackageErrParse && result.ErrorFile >= 0 && result.ErrorFile < len(graph.Packages[pkg].Files) {
					file := graph.Packages[pkg].Files[result.ErrorFile]
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
		renvoSetDiagnostic(&d, "checker", "RENVO-CHECK-001", "type checking failed")
		if built.Build.Error == build.BuildErrLower {
			renvoSetDiagnostic(&d, "lowerer", "RENVO-LOWER-001", "checked program could not be lowered")
		} else if built.Build.ErrorDetail >= check.CheckErrDuplicate && built.Build.ErrorDetail <= check.CheckErrCallArity {
			d.Code = renvoDiagnosticCode("CHECK", built.Build.ErrorDetail)
			d.Message = renvoCheckMessage(built.Build.ErrorDetail)
		}
	} else {
		renvoSetDiagnostic(&d, "linker", "RENVO-LINK-001", "package linking failed")
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
		d.Message += ": " + string(source.Src[d.Start:d.End])
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
