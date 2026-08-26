//go:build renvo

package lower

import "renvo.dev/internal/load"

func (b *coreUnitBuilder) addRTGAssembly(pkg load.Package) bool {
	if len(pkg.Assemblies) != 0 {
		b.errFile = len(pkg.Files)
		b.errToken = -1
		b.assemblyErrorPath = pkg.Assemblies[0].Path
		return false
	}
	if len(b.bodylessFuncs) != 0 {
		b.errFile = -1
		b.errToken = b.program.Funcs[b.bodylessFuncs[0]].NameTok
		return false
	}
	return true
}
