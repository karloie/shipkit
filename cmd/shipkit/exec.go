package main

import (
	"os/exec"
	"strings"
)

// ExecRunner abstracts os/exec so callers can be tested without real processes.
type ExecRunner interface {
	Run(name string, args ...string) error
	RunWithStdin(stdin, name string, args ...string) error
}

// defaultRunner is the package-level runner; tests swap it out.
var defaultRunner ExecRunner = &realExecRunner{}

type realExecRunner struct{}

func (r *realExecRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (r *realExecRunner) RunWithStdin(stdin, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.Run()
}
