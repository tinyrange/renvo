package driver

import (
	"fmt"
	"path"
	"renvo.dev/internal/elflink"
	"strings"
)

// ArchiveCommand implements deterministic ar r/c/s and ranlib in virtual memory.
// Args use ar's operation word followed by the archive and member filenames.
func ArchiveCommand(request *CommandRequest) (*CommandResult, error) {
	if request == nil || request.Filesystem == nil {
		return nil, fmt.Errorf("source filesystem must be specified")
	}
	if len(request.Args) < 2 {
		return linkFailure("", "ar requires operation and archive"), nil
	}
	flags := strings.TrimPrefix(request.Args[0], "-")
	replace, index := false, false
	for _, f := range flags {
		switch f {
		case 'r':
			replace = true
		case 's':
			index = true
		case 'c', 'D':
		default:
			return linkFailure("", "unsupported ar operation "+string(f)), nil
		}
	}
	if !replace && !index {
		return linkFailure("", "ar requires r or s"), nil
	}
	name := request.Args[1]
	var members []elflink.Input
	if data, ok := request.Filesystem.ReadFile(name); ok {
		var e elflink.Error
		members, e = elflink.ReadArchive(data)
		if e.Message != "" {
			return linkFailure(name, e.Message), nil
		}
	} else if !replace {
		return linkFailure(name, "could not read archive"), nil
	}
	if !replace && len(request.Args) > 2 {
		return linkFailure(name, "index operation does not accept members"), nil
	}
	for _, source := range request.Args[2:] {
		data, ok := request.Filesystem.ReadFile(source)
		if !ok {
			return linkFailure(source, "could not read archive member"), nil
		}
		member := elflink.Input{Name: path.Base(source), Data: data}
		found := false
		for i := range members {
			if members[i].Name == member.Name {
				members[i] = member
				found = true
				break
			}
		}
		if !found {
			members = append(members, member)
		}
	}
	data, e := elflink.WriteArchive(members)
	if e.Message != "" {
		return linkFailure(name, e.Message), nil
	}
	return commandResult(true, Diagnostic{}, name, data, "", nil), nil
}
