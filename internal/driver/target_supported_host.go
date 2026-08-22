//go:build !renvo

package driver

func renvoBackendTargetSupported(target string) bool {
	_ = target
	return false
}

func renvoBackendTargetBinding(target string) (string, string, int, bool) {
	_ = target
	return "", "", 0, false
}

func renvoBackendTargetHasBuildTag(target string, tag string) bool {
	_, _ = target, tag
	return false
}
