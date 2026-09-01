//go:build renvo_wasi_frontend && renvo_wasi_c_object

package main

import (
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

func renvoWasiCObjectCompile(args []string, workDir string, stdRoot string) (bool, int) {
	requested := false
	noStandardIncludes := false
	if len(args) >= 2 && args[1] == "cc" {
		for i := 2; i < len(args); i++ {
			if args[i] == "-c" || args[i] == "-mode=object" || args[i] == "-mode" && i+1 < len(args) && args[i+1] == "object" {
				requested = true
			}
			if args[i] == "-nostdinc" {
				noStandardIncludes = true
			}
		}
	}
	if !requested {
		return false, 0
	}
	if !noStandardIncludes {
		// A browser workspace stages the bundled C library beside stdRoot. The
		// ordinary object driver would otherwise search only host include roots,
		// which do not exist in WASI.
		withIncludes := make([]string, len(args), len(args)+2)
		copy(withIncludes, args)
		withIncludes = append(withIncludes, "-isystem", load.JoinPath(load.DirPath(stdRoot), "libc/include"))
		args = withIncludes
	}
	normalized := driver.NormalizeCCompilerCommand(args)
	built := driver.BuildFromFSOneShot(normalized[1:], workDir, stdRoot, driver.RenvoFS{})
	if !built.Ok {
		print(driver.FormatDiagnostic(built.Diagnostic))
		return true, 1
	}
	if !writeOutput(built.Options.Output, built.Unit) {
		print("renvo cc: could not write frontend output\n")
		return true, 1
	}
	return true, 0
}
