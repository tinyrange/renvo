//go:build rtg

package rtgx

import backend "j5.nz/rtg"

type backendError string

func (err backendError) Error() string {
	return string(err)
}

func compileSourceToBytes(source []byte, target string, backendRootOverride string, stripSymbols bool) ([]byte, error) {
	data, ok := backend.RtgCompileSourceToBytesStrip(source, target, stripSymbols)
	if !ok {
		return nil, backendError("rtg: compilation failed")
	}
	return data, nil
}

func compileSourceToOutput(source []byte, target string, backendRootOverride string, stripSymbols bool, output string) error {
	ok := backend.RtgCompileSourceToOutputStrip(source, target, output, stripSymbols)
	if !ok {
		return backendError("rtg: compilation failed")
	}
	return nil
}
