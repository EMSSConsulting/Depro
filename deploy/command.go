package deploy

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
	return "Deploy a version of code to your cluster"
}

// Help returns the help text for the deploy command
func (c *Command) Help() string {
	helpText := `
    Usage: depro deploy [options] version

        Deploys a specific version of code to the cluster

    Options:

        -server=127.0.0.1:8500 HTTP address of a Consul agent in the cluster
        -prefix=deploy/myapp
        -nodes=3
        -config=/etc/depro/myapp.json
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
		c.UI.Error(fmt.Sprintf("Failed to deploy '%s': %s", version, err.Error()))
		return 2
	}

	c.UI.Output(fmt.Sprintf("Version '%s' successfully deployed", version))
	return 0
}

func (c *Command) setupConfig() (string, error) {
	c.config = DefaultConfig()

	cmdFlags := flag.NewFlagSet("deploy", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	var configFile string
	cmdFlags.StringVar(&configFile, "config", "", "")

	if configFile != "" {
		cFile, err := ReadConfig(configFile)
		if err != nil {
			return "", err
		}

		c.config = Merge(c.config, cFile)
	}

	cmdFlags.StringVar(&c.config.Server, "server", "", "")
	cmdFlags.StringVar(&c.config.Prefix, "prefix", "", "")
	cmdFlags.IntVar(&c.config.Nodes, "nodes", 1, "")

	if err := cmdFlags.Parse(c.args); err != nil {
		return "", err
	}

	return cmdFlags.Arg(0), nil
}

func init() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	common.RegisterCommand("deploy", func() (cli.Command, error) {
		return &Command{
			UI: ui,
		}, nil
	})
}
