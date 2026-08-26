//go:build !renvo

package lower

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

func (b *coreUnitBuilder) addRTGAssembly(pkg load.Package) bool {
	if len(pkg.Assemblies) == 0 {
		return b.acceptNoRTGAssembly()
	}
	bound := make([]bool, len(b.program.Funcs))
	for fileIndex := 0; fileIndex < len(pkg.Assemblies); fileIndex++ {
		file := pkg.Assemblies[fileIndex]
		document := parseRTGAssemblyBindings(file.Src)
		if !document.ok {
			return b.failRTGAssembly(pkg, fileIndex, document.errorOffset)
		}
		path := file.Path
		if len(path) > len(pkg.Ref.Dir) && path[:len(pkg.Ref.Dir)] == pkg.Ref.Dir && path[len(pkg.Ref.Dir)] == '/' {
			path = path[len(pkg.Ref.Dir)+1:]
		}
		sourceIndex := len(b.program.RTGAssembly)
		source := append([]byte(nil), file.Src...)
		b.program.RTGAssembly = append(b.program.RTGAssembly, unit.RTGAssemblySource{Path: cloneCoreString(path), Source: source})
		for entryIndex := 0; entryIndex < len(document.entries); entryIndex++ {
			entry := document.entries[entryIndex]
			function := -1
			for i := 0; i < len(b.program.Funcs); i++ {
				fn := b.program.Funcs[i]
				if fn.ReceiverStart == fn.ReceiverEnd && fn.NameEnd-fn.NameStart == len(entry.name) &&
					string(b.program.Text[fn.NameStart:fn.NameEnd]) == entry.name {
					function = i
					break
				}
			}
			if function < 0 || bound[function] || b.program.Funcs[function].BodyEnd != b.program.Funcs[function].BodyStart {
				return b.failRTGAssembly(pkg, fileIndex, entry.offset)
			}
			bound[function] = true
			b.program.RTGAssemblyFuncs = append(b.program.RTGAssemblyFuncs, unit.RTGAssemblyBinding{Func: function, Source: sourceIndex, Entry: entryIndex})
		}
	}
	for i := 0; i < len(b.bodylessFuncs); i++ {
		if !bound[b.bodylessFuncs[i]] {
			return b.failBodylessFunction(b.bodylessFuncs[i])
		}
	}
	return true
}

func (b *coreUnitBuilder) acceptNoRTGAssembly() bool {
	return len(b.bodylessFuncs) == 0 || b.failBodylessFunction(b.bodylessFuncs[0])
}

func (b *coreUnitBuilder) failBodylessFunction(function int) bool {
	b.errFile = -1
	b.errToken = b.program.Funcs[function].NameTok
	return false
}

func (b *coreUnitBuilder) failRTGAssembly(pkg load.Package, fileIndex int, offset int) bool {
	b.errFile = len(pkg.Files) + fileIndex
	b.errToken = -1
	b.assemblyErrorPath = pkg.Assemblies[fileIndex].Path
	b.assemblyErrorOffset = offset
	return false
}
