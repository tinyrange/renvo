//go:build renvo

package driver

import "renvo.dev/internal/unit"

func resolveFSBuildSessionOptions(args []string, workDir string, fs SourceFS) (Options, unit.TargetBinding) {
	return parseFSOptions(args, workDir, fs), unit.TargetBinding{}
}
