//go:build !linux && !darwin && !windows

package testmeasure

import (
	"fmt"
	"os/exec"
)

type observation struct{}

func prepare(cmd *exec.Cmd) (*observation, error) {
	return nil, fmt.Errorf("process accounting is unsupported on this host")
}
func (*observation) started(cmd *exec.Cmd) error                { return nil }
func (*observation) close()                                     {}
func (*observation) finish(cmd *exec.Cmd, result *Result) error { return nil }
