package rtg

import "renvo.dev/internal/rtgprofile"

// SystemProfile is the hosted resource contract shared by the definition
// language and the frontend driver.
type SystemProfile struct {
	Name        string
	Target      string
	BinaryLimit int
	ArenaSize   int
}

func ParseSystem(source []byte, filename string) (SystemProfile, Diagnostic, bool) {
	parsed, diagnostic, ok := rtgprofile.Parse(source)
	if !ok {
		return SystemProfile{}, Diagnostic{
			Filename: filename,
			Span:     sourceSpan(source, diagnostic.Offset, diagnostic.Offset),
			Code:     "RTG-SYSTEM-001",
			Message:  diagnostic.Message,
		}, false
	}
	return SystemProfile{
		Name:        parsed.Name,
		Target:      parsed.Target,
		BinaryLimit: parsed.BinaryLimit,
		ArenaSize:   parsed.ArenaSize,
	}, Diagnostic{}, true
}
