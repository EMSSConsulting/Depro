package agent

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/common"
	"github.com/EMSSConsulting/Depro/util"
	"github.com/mitchellh/cli"
)

// Command is a command implementation which runs an agent on the local
// node to perform deployments.
type Command struct {
	UI     cli.Ui
	config *Config
	args   []string
}

// Synopsis returns a short summary of the command
func (c *Command) Synopsis() string {
	return "Run a deployment agent on the local node"
}

// Help returns the help text for the deploy command
func (c *Command) Help() string {
	helpText := `
    Usage: depro agent [options]

        Runs an agent which will execute deployments on the local node

    Options:

        -server=127.0.0.1:8500 HTTP address of a Consul agent in the cluster
        -config-dir=/etc/depro/
        -config-file=/etc/depro/myapp.json
    `

	return strings.TrimSpace(helpText)
}

// Run executes the deployment command
func (c *Command) Run(args []string) int {
	c.args = args
	err := c.setupConfig()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	go func() {
		select {
		case <-util.MakeShutdownCh():
			c.UI.Info("Shutting down agents, waiting for inflight requests and running tasks to complete.")
		}

		select {
		case <-util.MakeShutdownCh():
			c.UI.Info("Forcing inflight requests to complete and exiting running tasks.")
			os.Exit(1)
		}
	}()

	agent := NewOperation(c.UI, c.config)

	err = agent.Run()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to run agent: %s", err.Error()))
		return 2
	}

	return 0
}

func (c *Command) setupConfig() error {
	c.config = DefaultConfig()

	cmdFlags := flag.NewFlagSet("agent", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	var configFiles []string
	cmdFlags.Var((*util.AppendSliceValue)(&configFiles), "config-dir", "directory of json files to read")
	cmdFlags.Var((*util.AppendSliceValue)(&configFiles), "config-file", "json file to read config from")

	cmdFlags.StringVar(&c.config.Server, "server", "", "")

	if err := cmdFlags.Parse(c.args); err != nil {
		return err
	}

	if len(configFiles) > 0 {
		cFile, err := ReadConfig(configFiles)
		if err != nil {
			return err
		}

		c.config = Merge(c.config, cFile)
	}

	return nil
}

func init() {
	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	common.RegisterCommand("agent", func() (cli.Command, error) {
		return &Command{
			UI: ui,
		}, nil
	})
}
