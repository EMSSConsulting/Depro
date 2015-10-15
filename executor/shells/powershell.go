package shells

import (
	"fmt"
	"path"

	"github.com/EMSSConsulting/Depro/executor"
)

// Powershell provides a shell instance which will run tasks using the Windows
// PowerShell interpreter.
type Powershell struct {
	ShellAbstract
}

// Name retrieves the name of this shell as it will be used when requesting a new
// executor instance.
func (p *Powershell) Name() string {
	return "powershell"
}

// Command retrieves the relative or full path to the executable which will be run
// for this shell.
// If a relative path is provided, the user's PATH environment variable will be
// searched in an attempt to locate the file to execute.
func (p *Powershell) Command() string {
	return "powershell.exe"
}

// Filename is responsible for building a valid filename from the taskID and the
// temporary task directory.
func (p *Powershell) Filename(directory, taskID string) string {
	return path.Join(directory, taskID+".ps1")
}

// Args is responsible for building the full list of command line arguments
// which will be passed to the shell's executable.
// It should include the file to be run, which can be acquired by calling
// `task.ScriptFile(s)`.
func (p *Powershell) Args(executor *executor.Executor, task *executor.Task) []string {
	return append([]string{
		"-noprofile",
		"-noninteractive",
		"-executionpolicy",
		"Bypass",
		"-command",
		task.ScriptFile(p),
	}, task.Args...)
}

// Operations is responsible for building the list of operations that will
// be included in the task script.
func (p *Powershell) Operations(executor *executor.Executor, task *executor.Task) []string {
	ops := make([]string, 0, len(executor.Environment)+len(task.Environment)+2)

	ops = append(ops, "# Environment Variables")

	for key, value := range executor.Environment {
		ops = append(ops, fmt.Sprintf("$%s='%s'", key, value))
	}

	for key, value := range task.Environment {
		ops = append(ops, fmt.Sprintf("$%s='%s'", key, value))
	}

	ops = append(ops, "# Task Operations")

	return append(ops, task.Operations...)
}

func init() {
	executor.RegisterShell(&Powershell{})
}
