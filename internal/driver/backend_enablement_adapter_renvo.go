//go:build renvo

package driver

import "renvo.dev/internal/rbe"

func backendEnablementFSForBuild(base SourceFS, _ string, _ []rbe.File) SourceFS { return base }
