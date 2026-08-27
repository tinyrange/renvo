//go:build renvo && !renvo_wasi_c_object

package c11

// The self-hosting compiler keeps its original compact header path. Browser
// C object builds opt into the active-header implementation.
func BuildObjectPrelude(path string, src []byte, reader IncludeReader, headers ...HeaderSource) HeaderResult {
	result := HeaderResult{Ok: true, ErrorAt: -1}
	includes := sourceIncludes(src)
	if len(includes) == 0 {
		return result
	}
	wanted := sourceCallNames(src)
	var emitted []string
	for i := 0; i < len(includes); i++ {
		header, headerPath, ok := reader.ReadInclude(path, includes[i].name, includes[i].angled)
		if !ok {
			return HeaderResult{Ok: false, ErrorPath: includes[i].name, ErrorAt: includes[i].at}
		}
		if findText(result.Dependencies, headerPath) < 0 {
			result.Dependencies = append(result.Dependencies, headerPath)
		}
		declarations, names, ok := headerDeclarations(header, wanted, emitted)
		if !ok {
			return HeaderResult{Ok: false, ErrorPath: headerPath, ErrorAt: includes[i].at}
		}
		result.Prelude = append(result.Prelude, declarations...)
		emitted = append(emitted, names...)
	}
	return result
}
