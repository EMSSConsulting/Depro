package executor

import (
	"fmt"

	"github.com/satori/go.uuid"
)

import "os"

// Task represents a single task which should be run by an executor instance.
// This task should include the list of operations to run and may include
// custom environment variables, additional arguments and other options.
type Task struct {
	Operations []string

	Args        []string
	Environment map[string]string
	Directory   string

	uuid string
}

// NewTask creates a task which can be executed by an executor to perform
// a set of operations.
func NewTask(operations, args []string, environment map[string]string) (*Task, error) {
	if operations == nil {
		operations = []string{}
	}

	if args == nil {
		args = []string{}
	}

	if environment == nil {
		environment = map[string]string{}
	}

	if len(operations) == 0 {
		return nil, fmt.Errorf("Expected at least one operation to be specified for this task.")
	}

	task := &Task{
		Operations:  operations,
		Args:        args,
		Environment: environment,
		uuid:        uuid.NewV4().String(),
	}

	return task, nil
}

// ScriptFile retrieves the full path of the file in which the script for this
// task will be stored during execution of the task.
func (t *Task) ScriptFile(shell Shell) string {
	return shell.Filename(os.TempDir(), t.uuid)
}
