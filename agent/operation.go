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
		d := NewDeployment(o, &deployment)

		go func() {
			o.UI.Info(fmt.Sprintf("Starting agent '%s'", d.Config.ID))
			err := d.Run()
			if err != nil {
				o.UI.Error(fmt.Sprintf("Failed to run agent '%s': %s", d.Config.ID, err))
			} else {
				o.UI.Info(fmt.Sprintf("Stopping agent '%s'", d.Config.ID))
			}
		}()
	}

	return nil
}
