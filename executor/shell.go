package executor

import "fmt"

var shells map[string]Shell

// Shell represents the logic necessary to prepare a list of operations for execution
// on a specific operating system command shell.
// It is important to note that the majority of operations are not portable across
// shell interpreters, so you will need to ensure you use the correct shell for
// the operations you will be providing.
type Shell interface {
	Name() string
	Command() string
	Filename(directory, taskID string) string
	Environment(executor *Executor, task *Task) []string
	Operations(executor *Executor, task *Task) []string
	Args(executor *Executor, task *Task) []string
	JoinOperations(operations []string) string
}

// RegisterShell allows you to register your own custom shell implementations
// with executor.
func RegisterShell(instance Shell) {
	if shells == nil {
		shells = map[string]Shell{}
	}

	_, exists := shells[instance.Name()]
	if exists {
		panic(fmt.Sprintf("Shell '%s' has already been registered with executor.", instance.Name()))
	}

	shells[instance.Name()] = instance
}

// GetShell retrieves a shell with the given name, or nil if there is not a
// shell with that name registered.
func GetShell(name string) Shell {
	if shells == nil {
		return nil
	}

	return shells[name]
}
