package query

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/common"
	"github.com/mitchellh/cli"
)

// Command is a command implementation which manages the deployment process
// for a version of code.
type Command struct {
	UI     cli.Ui
	config *Config
	args   []string
}

// Synopsis returns a short summary of the command
func (c *Command) Synopsis() string {
	return "Query your cluster to determine its current state"
}

// Help returns the help text for the deploy command
func (c *Command) Help() string {
	helpText := `
    Usage: depro query [options] [version]

        Gets the list of nodes and their status for the current, or specified, version

    Options:

        -server=127.0.0.1:8500 HTTP address of a Consul agent in the cluster
        -prefix=deploy/myapp
        -config=/etc/depro/myapp.json
		-auth=username:password
    `

	return strings.TrimSpace(helpText)
}

// Run executes the deployment command
func (c *Command) Run(args []string) int {
	c.args = args
	version, err := c.setupConfig()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	op := NewOperation(c.UI, c.config, version)

	err = op.Run()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to query '%s': %s", version, err.Error()))
		return 2
	}

	return 0
}

func (c *Command) setupConfig() (string, error) {
	c.config = DefaultConfig()

	cmdFlags := flag.NewFlagSet("query", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := ParseFlags(c.config, c.args, cmdFlags)
	if err != nil {
		return "", err
	}

	return cmdFlags.Arg(0), nil
}

func init() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	common.RegisterCommand("query", func() (cli.Command, error) {
		return &Command{
			UI: ui,
		}, nil
	})
}
