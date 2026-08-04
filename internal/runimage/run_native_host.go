//go:build !renvo && ((!linux && !windows && !darwin) || (windows && !amd64 && !386 && !arm64) || (darwin && !arm64))

package runimage

import (
	"fmt"
	"io"
	"os/exec"

	"renvo.dev/internal/linkedimage"
)

func runNative(image linkedimage.Image, script string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) Result {
	prepared, err := prepareNativeExecutable(image.Native)
	if err != nil {
		return Result{ExitCode: 1, Err: err}
	}
	defer prepared.close()
	command := exec.Command(prepared.path, args...)
	command.Args[0] = script
	command.ExtraFiles = prepared.extraFiles
	command.Env = env
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	err = command.Run()
	if err == nil {
		return Result{Loader: prepared.loader}
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return Result{ExitCode: exit.ExitCode(), Loader: prepared.loader}
	}
	return Result{ExitCode: 1, Loader: prepared.loader, Err: fmt.Errorf("start linked image: %w", err)}
}
