//go:build rtg

package rtgx

import backend "j5.nz/rtg"

type backendError string

func (err backendError) Error() string {
	return string(err)
}

func compileSourceToBytes(source []byte, target string, backendRootOverride string) ([]byte, error) {
	data, ok := backend.RtgCompileSourceToBytes(source, target)
	if !ok {
		return nil, backendError("rtg: compilation failed")
	}
	return data, nil
}
