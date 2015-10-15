package version

import (
	"bytes"
	"fmt"
	"os"

	"github.com/EMSSConsulting/Depro/common"
	"github.com/mitchellh/cli"
)

var (
	GitCommit string
	Version   string
	Release   string
)

// Command is a command which prints the application version
type Command struct {
	Revision          string
	Version           string
	VersionPrerelease string
	UI                cli.Ui
}

// Help prints command specific help information
func (c *Command) Help() string {
	return ""
}

// Run executes the command and returns an exit code
func (c *Command) Run(_ []string) int {
	var versionString bytes.Buffer
	fmt.Fprintf(&versionString, "Depro %s", c.Version)
	if c.Revision != "" {
		fmt.Fprintf(&versionString, " (%s)", c.Revision)
	}

	c.UI.Output(versionString.String())
	return 0
}

// Synopsis prints a short description of the command
func (c *Command) Synopsis() string {
	return "Prints the Depro version"
}

func init() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	common.RegisterCommand("version", func() (cli.Command, error) {
		return &Command{
			Revision:          GitCommit,
			Version:           Version,
			VersionPrerelease: Release,
			UI:                ui,
		}, nil
	})
}
