//go:build rtg

package rtgx

type backendError string

func (err backendError) Error() string {
	return string(err)
}

func compileSourceToBytes(source []byte, target string, backendRootOverride string) ([]byte, error) {
	return nil, backendError("rtg: in-memory rtgx backend is not implemented")
}
