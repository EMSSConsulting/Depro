package shells

import (
	"path"

	"github.com/EMSSConsulting/Depro/executor"
)

// CommandPrompt provides a shell instance which will run tasks using the Windows
// cmd interpreter.
type CommandPrompt struct {
	ShellAbstract
}

func init() {
	executor.RegisterShell(&CommandPrompt{})
}

// Name retrieves the name of this shell as it will be used when requesting a new
// executor instance.
func (c *CommandPrompt) Name() string {
	return "cmd"
}

// Command retrieves the relative or full path to the executable which will be run
// for this shell.
// If a relative path is provided, the user's PATH environment variable will be
// searched in an attempt to locate the file to execute.
func (c *CommandPrompt) Command() string {
	return "cmd.exe"
}

// Filename is responsible for building a valid filename from the taskID and the
// temporary task directory.
func (c *CommandPrompt) Filename(directory, taskID string) string {
	return path.Join(directory, taskID+".bat")
}

// Args is responsible for building the full list of command line arguments
// which will be passed to the shell's executable.
// It should include the file to be run, which can be acquired by calling
// `task.ScriptFile(s)`.
func (c *CommandPrompt) Args(executor *executor.Executor, task *executor.Task) []string {
	return append([]string{"/Q", "/C", task.ScriptFile(c)}, task.Args...)
}
