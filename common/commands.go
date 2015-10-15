package common

import "github.com/mitchellh/cli"

var commands map[string]cli.CommandFactory

func RegisterCommand(name string, factory cli.CommandFactory) {
	if commands == nil {
		commands = map[string]cli.CommandFactory{}
	}

	commands[name] = factory
}

func Commands() map[string]cli.CommandFactory {
	if commands == nil {
		panic("No commands registered yet with the application")
	}

	return commands
}
