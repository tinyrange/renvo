//go:build !renvo

package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/rbe"
)

func backendEnablementFSForBuild(base SourceFS, stdRoot string, files []rbe.File) SourceFS {
	return backendEnablementFS{base: base, stdRoot: load.CleanPath(stdRoot), files: files}
}
