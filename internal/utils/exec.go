package utils

import (
	"bytes"
	"os/exec"
)

func Exec(command ...string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	var out bytes.Buffer
	var errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout

	err := cmd.Run()
	if err != nil {
		// Return stderr for error diagnostics
		return errout.String(), err
	}

	return out.String(), nil
}

// ExecWithStdin executes a command with stdin input
func ExecWithStdin(command []string, stdin string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	var out bytes.Buffer
	var errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	cmd.Stdin = bytes.NewBufferString(stdin)

	err := cmd.Run()
	if err != nil {
		// Return stderr output for debugging
		return errout.String(), err
	}

	return out.String(), nil
}
