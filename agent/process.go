package agent

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// Operation contains the configuration and clients for performing a deployment
type Operation struct {
	UI     cli.Ui
	Config *Config
	Client *api.Client
}

// Run executes the process for a deployment operation
func (o *Operation) Run() error {
	for _, deployment := range o.Config.Deployments {
		deployment.agent = o

		go func(deployment *Deployment) {
			o.UI.Info(fmt.Sprintf("Starting agent '%s'", deployment.ID))
			err := deployment.Run()
			if err != nil {
				o.UI.Error(fmt.Sprintf("Failed to run agent '%s': %s", deployment.ID, err))
			} else {
				o.UI.Info(fmt.Sprintf("Stopping agent '%s'", deployment.ID))
			}
		}(&deployment)
	}

	return nil
}
