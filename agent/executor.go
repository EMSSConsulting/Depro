package agent

import (
	"os"
	"os/exec"
)

type Task struct {
	Command     string
	Environment []string
}

type Executor struct {
	Environment []string
	Directory   string
}

func (e *Executor) Run(task *Task) error {
	cmd := exec.Command(task.Command)

	cmd.Dir = e.Directory

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, e.Environment...)
	cmd.Env = append(cmd.Env, task.Environment...)

	return cmd.Run()
}
