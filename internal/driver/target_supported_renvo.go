//go:build renvo

package driver

import "renvo.dev/internal/backendbridge"

func renvoBackendTargetSupported(target string) bool {
	return backendbridge.TargetSupported(target)
}

func renvoBackendTargetBinding(target string) (string, string, int, bool) {
	return backendbridge.TargetBinding(target)
}

func renvoBackendTargetHasBuildTag(target string, tag string) bool {
	return backendbridge.TargetHasBuildTag(target, tag)
}
