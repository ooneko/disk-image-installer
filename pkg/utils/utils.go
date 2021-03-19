package utils

import (
	"bytes"
	"os/exec"
)

func RunCommand(command string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
