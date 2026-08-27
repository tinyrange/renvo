//go:build renvo

package driver

import (
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/unit"
)

func resolveFSBuildSessionOptions(args []string, workDir string, fs SourceFS) (Options, unit.TargetBinding, []rbe.File) {
	return parseFSOptions(args, workDir, fs), unit.TargetBinding{}, nil
}
