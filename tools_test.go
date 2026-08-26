package main

import (
	"errors"
	"os/exec"
	"testing"
)

func TestExec(t *testing.T) {
	cmd := exec.Command("/usr/bin/echo", "OK")
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	out, err := cmd.Output()
	if err != nil {
		t.Log("could not run command: ", err)
	}

	t.Log(string(out))
	if string(out) != "OK\n" {
		t.Fatal("Command output do not match.")
	}
}
