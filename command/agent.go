package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/agent"
	"github.com/EMSSConsulting/Depro/util"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// AgentCommand is a command implementation which runs an agent on the local
// node to perform deployments.
type AgentCommand struct {
	UI     cli.Ui
	config *agent.Config
	args   []string
}

// Help returns the help text for the deploy command
func (c *AgentCommand) Help() string {
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

func (c *AgentCommand) setupConfig() error {
	c.config = agent.DefaultConfig()

	cmdFlags := flag.NewFlagSet("agent", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	var configFiles []string
	cmdFlags.Var((*AppendSliceValue)(&configFiles), "config-dir", "directory of json files to read")
	cmdFlags.Var((*AppendSliceValue)(&configFiles), "config-file", "json file to read config from")

	cmdFlags.StringVar(&c.config.Server, "server", "", "")

	if err := cmdFlags.Parse(c.args); err != nil {
		return err
	}

	if len(configFiles) > 0 {
		cFile, err := agent.ReadConfig(configFiles)
		if err != nil {
			return err
		}

		c.config = agent.Merge(c.config, cFile)
	}

	return nil
}

func (c *AgentCommand) runAgent() error {
	apiConfig := api.DefaultConfig()

	apiConfig.Address = c.config.Server
	apiConfig.WaitTime = c.config.WaitTime

	client, _ := api.NewClient(apiConfig)

	agent := agent.Operation{
		Client: client,
		Config: c.config,
		UI:     c.UI,
	}

	return agent.Run()
}

// Run executes the deployment command
func (c *AgentCommand) Run(args []string) int {
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

	err = c.runAgent()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to run agent: %s", err.Error()))
		return 2
	}

	return 0
}

// Synopsis returns a short summary of the command
func (c *AgentCommand) Synopsis() string {
	return "Run a deployment agent on the local node"
}
