//go:build renvo_wasi_frontend && !renvo_wasi_c_object

package main

func renvoWasiCObjectCompile(args []string, workDir string, stdRoot string) (bool, int) {
	return false, 0
}
