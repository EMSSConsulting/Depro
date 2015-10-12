package command

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/cli"
)

// VersionCommand is a command which prints the application version
type VersionCommand struct {
	Revision          string
	Version           string
	VersionPrerelease string
	UI                cli.Ui
}

// Help prints command specific help information
func (c *VersionCommand) Help() string {
	return ""
}

// Run executes the command and returns an exit code
func (c *VersionCommand) Run(_ []string) int {
	var versionString bytes.Buffer
	fmt.Fprintf(&versionString, "Depro %s", c.Version)
	if c.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, ".%s", c.VersionPrerelease)

		if c.Revision != "" {
			fmt.Fprintf(&versionString, " (%s)", c.Revision)
		}
	}

	c.UI.Output(versionString.String())
	return 0
}

// Synopsis prints a short description of the command
func (c *VersionCommand) Synopsis() string {
	return "Prints the Depro version"
}
