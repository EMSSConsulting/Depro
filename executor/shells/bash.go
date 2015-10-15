package shells

import (
	"path"

	"github.com/EMSSConsulting/Depro/executor"
)

// Bash provides a shell instance which will run tasks using the Unix bash shell.
type Bash struct {
	ShellAbstract
}

func init() {
	executor.RegisterShell(&Bash{})
}

// Name retrieves the name of this shell as it will be used when requesting a new
// executor instance.
func (b *Bash) Name() string {
	return "bash"
}

// Command retrieves the relative or full path to the executable which will be run
// for this shell.
// If a relative path is provided, the user's PATH environment variable will be
// searched in an attempt to locate the file to execute.
func (b *Bash) Command() string {
	return "bash"
}

// Filename is responsible for building a valid filename from the taskID and the
// temporary task directory.
func (b *Bash) Filename(directory, taskID string) string {
	return path.Join(directory, taskID+".sh")
}

// Args is responsible for building the full list of command line arguments
// which will be passed to the shell's executable.
// It should include the file to be run, which can be acquired by calling
// `task.ScriptFile(s)`.
func (b *Bash) Args(executor *executor.Executor, task *executor.Task) []string {
	return append([]string{task.ScriptFile(b)}, task.Args...)
}

// Operations is responsible for building the list of operations that will
// be included in the task script.
func (b *Bash) Operations(executor *executor.Executor, task *executor.Task) []string {
	return append([]string{"#!/usr/bin/env bash"}, task.Operations...)
}
