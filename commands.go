package main

import (
	"os"

	"github.com/EMSSConsulting/Depro/command"
	"github.com/mitchellh/cli"
)

// CLI commands to be made available to users
var Commands map[string]cli.CommandFactory

func init() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	Commands = map[string]cli.CommandFactory{
		"version": func() (cli.Command, error) {
			ver := Version
			rel := VersionPrerelease

			if GitDescribe != "" {
				ver = GitDescribe
			}

			if GitDescribe == "" && rel == "" {
				rel = "dev"
			}

			return &command.VersionCommand{
				Revision:          GitCommit,
				Version:           ver,
				VersionPrerelease: rel,
				UI:                ui,
			}, nil
		},

		"deploy": func() (cli.Command, error) {
			return &command.DeployCommand{
				UI: ui,
			}, nil
		},

		"agent": func() (cli.Command, error) {
			return &command.AgentCommand{
				UI: ui,
			}, nil
		},
	}
}
