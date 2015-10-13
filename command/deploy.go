package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/EMSSConsulting/Depro/deploy"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// DeployCommand is a command implementation which manages the deployment process
// for a version of code.
type DeployCommand struct {
	UI     cli.Ui
	config *deploy.Config
	args   []string
}

// Help returns the help text for the deploy command
func (c *DeployCommand) Help() string {
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

func (c *DeployCommand) setupConfig() (string, error) {
	c.config = deploy.DefaultConfig()

	cmdFlags := flag.NewFlagSet("deploy", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	var configFile string
	cmdFlags.StringVar(&configFile, "config", "", "")

	if configFile != "" {
		cFile, err := deploy.ReadConfig(configFile)
		if err != nil {
			return "", err
		}

		c.config = deploy.Merge(c.config, cFile)
	}

	cmdFlags.StringVar(&c.config.Server, "server", "", "")
	cmdFlags.StringVar(&c.config.Prefix, "prefix", "", "")
	cmdFlags.IntVar(&c.config.Nodes, "nodes", 1, "")

	if err := cmdFlags.Parse(c.args); err != nil {
		return "", err
	}

	return cmdFlags.Arg(0), nil
}

func (c *DeployCommand) deployVersion(version string) error {
	apiConfig := api.DefaultConfig()

	apiConfig.Address = c.config.Server
	apiConfig.WaitTime = c.config.WaitTime

	client, _ := api.NewClient(apiConfig)

	deploy := deploy.Operation{
		Version: version,
		Client:  client,
		Config:  c.config,
	}

	return deploy.Run()
}

// Run executes the deployment command
func (c *DeployCommand) Run(args []string) int {
	c.args = args
	version, err := c.setupConfig()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = c.deployVersion(version)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to deploy '%s': %s", version, err.Error()))
		return 2
	}

	c.UI.Output(fmt.Sprintf("Version '%s' successfully deployed", version))
	return 0
}

// Synopsis returns a short summary of the command
func (c *DeployCommand) Synopsis() string {
	return "Deploy a version of code to your cluster"
}
