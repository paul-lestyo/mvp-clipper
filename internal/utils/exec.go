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
		return "", err
	}

	return out.String(), nil
}
