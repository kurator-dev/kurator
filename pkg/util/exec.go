package util

import (
	"os"
	"os/exec"
)

// RunCommand executes the given cmd, and streaming the realtime output.
func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
