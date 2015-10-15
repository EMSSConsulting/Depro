package executor

import (
	"io/ioutil"
	"os"
	"os/exec"
)

// Executor is responsible for running various tasks using the configured shell.
type Executor struct {
	Environment map[string]string
	Directory   string
	Shell       Shell
}

// NewExecutor is responsible for creating a new execution context which makes
// use of the given shell to execute tasks.
func NewExecutor(shell string) Executor {
	sh := GetShell(shell)

	cwd, _ := os.Getwd()

	return Executor{
		Shell:       sh,
		Directory:   cwd,
		Environment: map[string]string{},
	}
}

// Run executes the given task and returns an error if the final exit code is
// non-zero or execution fails for another reason.
func (e *Executor) Run(task *Task) error {
	defer e.cleanup(task)

	err := e.prepare(task)
	if err != nil {
		return err
	}

	cmd := e.getCommand(task)

	return cmd.Run()
}

// RunOutput executes the given task and returns the combined Stdout and Stderr
// of the process, as well as an error if the final exit code is
// non-zero or execution fails for another reason.
func (e *Executor) RunOutput(task *Task) ([]byte, error) {
	defer e.cleanup(task)

	err := e.prepare(task)
	if err != nil {
		return nil, err
	}

	cmd := e.getCommand(task)

	return cmd.CombinedOutput()
}

func (e *Executor) prepare(task *Task) error {
	operations := e.Shell.Operations(e, task)
	file := task.ScriptFile(e.Shell)

	err := ioutil.WriteFile(file, []byte(e.Shell.JoinOperations(operations)), 0)
	if err != nil {
		return err
	}

	return nil
}

func (e *Executor) cleanup(task *Task) {
	file := task.ScriptFile(e.Shell)

	os.Remove(file)
}

func (e *Executor) getCommand(task *Task) *exec.Cmd {
	cmd := exec.Command(e.Shell.Command(), e.Shell.Args(e, task)...)

	cmd.Dir = e.Directory

	if task.Directory != "" {
		cmd.Dir = task.Directory
	}

	cmd.Env = e.Shell.Environment(e, task)

	return cmd
}
