package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Task struct {
	Arguments    []string
	Environment  []string
	Instructions []string
}

type Executor struct {
	Command              string
	Arguments            []string
	Extension            string
	InstructionSeparator string
	Environment          []string
	Directory            string
}

func (e *Executor) Run(task *Task) (string, error) {
	scriptFile, err := e.prepareScript(task)
	if err != nil {
		return "", err
	}

	defer os.Remove(scriptFile)

	args := append(e.Arguments, task.Arguments...)
	args = append(args, scriptFile)

	cmd := exec.Command(e.Command, args...)

	cmd.Dir = e.Directory

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, e.Environment...)
	cmd.Env = append(cmd.Env, task.Environment...)

	out, err := cmd.CombinedOutput()
	return string(out), nil
}

func (e *Executor) prepareScript(task *Task) (string, error) {
	file, err := ioutil.TempFile("", "depro_")
	if err != nil {
		return "", err
	}

	defer os.Remove(file.Name())

	script := strings.Join(task.Instructions, e.InstructionSeparator)

	err = ioutil.WriteFile(file.Name(), []byte(script), 0)
	if err != nil {
		os.Remove(file.Name())
		return "", nil
	}

	err = file.Close()
	if err != nil {
		return "", nil
	}

	newFileName := fmt.Sprintf("%s.%s", file.Name(), strings.Trim(e.Extension, "."))
	os.Rename(file.Name(), newFileName)

	return newFileName, nil
}
