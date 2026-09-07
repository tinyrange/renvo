package load

// CompilerGoVersion is the language baseline advertised by Renvo's release
// build tags. Keep it explicit so selection does not depend on the host Go
// compiler, and separate from each module's requested language version.
// This baseline follows the repository's Go 1.25 bootstrap requirement;
// accepting newer module metadata does not advertise newer release tags.
const CompilerGoVersion = "1.25"

// HasGoReleaseTag recognizes canonical release tags through our baseline.
// Patch, prerelease, and leading-zero spellings are ordinary user tags.
func HasGoReleaseTag(tag string) bool {
	if len(tag) < 5 || tag[:4] != "go1." || tag[4] == '0' {
		return false
	}
	for i := 4; i < len(tag); i++ {
		if tag[i] < '0' || tag[i] > '9' {
			return false
		}
	}
	return !GoVersionBefore(CompilerGoVersion, tag[2:])
}
