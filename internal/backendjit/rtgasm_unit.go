package backendjit

import "renvo.dev/internal/unit"

func attachRTGAssemblyCode(data []byte, code [][]byte) ([]byte, bool) {
	_, bindings, ok := unit.ReadRTGAssemblyFragments(data)
	if !ok {
		return nil, false
	}
	return unit.AttachRTGAssemblyFragments(data, bindings, code)
}
func attachRTGAssemblyFragments(data []byte, bindings []unit.RTGAssemblyBinding, code [][]byte) ([]byte, bool) {
	return unit.AttachRTGAssemblyFragments(data, bindings, code)
}
func readRTGAssembly(data []byte) ([]unit.RTGAssemblySource, []unit.RTGAssemblyBinding, bool) {
	return unit.ReadRTGAssemblyFragments(data)
}
