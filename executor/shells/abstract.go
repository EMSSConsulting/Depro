package shells

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/EMSSConsulting/Depro/executor"
)

// ShellAbstract represents a shell which can be used by executor to run scripts.
// It is responsible for setting up the scripts that are run, as well as the environment
// that they are run under.
type ShellAbstract struct {
}

// Name retrieves the name of this shell as it will be used when requesting a new
// executor instance.
func (s *ShellAbstract) Name() string {
	return "abstract"
}

// Command retrieves the relative or full path to the executable which will be run
// for this shell.
// If a relative path is provided, the user's PATH environment variable will be
// searched in an attempt to locate the file to execute.
func (s *ShellAbstract) Command() string {
	panic("You have not implemented this shell's Command() method.")
}

// Filename is responsible for building a valid filename from the taskID and the
// temporary task directory.
func (s *ShellAbstract) Filename(directory, taskID string) string {
	return path.Join(directory, taskID)
}

// Environment is responsible for building the full execution environment
// under which this task should be run.
func (s *ShellAbstract) Environment(executor *executor.Executor, task *executor.Task) []string {
	env := make([]string, 0, len(executor.Environment)+len(task.Environment))

	for key, value := range executor.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	for key, value := range task.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return append(os.Environ(), env...)
}

// Operations is responsible for building the list of operations that will
// be included in the task script.
func (s *ShellAbstract) Operations(executor *executor.Executor, task *executor.Task) []string {
	return task.Operations
}

// Args is responsible for building the full list of command line arguments
// which will be passed to the shell's executable.
// It should include the file to be run, which can be acquired by calling
// `task.ScriptFile(s)`.
func (s *ShellAbstract) Args(executor *executor.Executor, task *executor.Task) []string {
	return append([]string{task.ScriptFile(s)}, task.Args...)
}

// JoinOperations is responsible for joining the list of operations into
// a single string which can be written to a script file.
func (s *ShellAbstract) JoinOperations(operations []string) string {
	return strings.Join(operations, "\n")
}
