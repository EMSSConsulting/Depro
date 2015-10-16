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

func NewOperation(ui cli.Ui, config *Config) Operation {
	apiConfig := api.DefaultConfig()

	apiConfig.Address = config.Server
	apiConfig.Datacenter = config.Datacenter
	apiConfig.WaitTime = config.WaitTime

	client, _ := api.NewClient(apiConfig)

	return Operation{
		Client: client,
		Config: config,
		UI:     ui,
	}
}

// Run executes the process for a deployment operation
func (o *Operation) Run() error {
	shutdownCh := make(chan struct{})

	for _, deployment := range o.Config.Deployments {
		d := NewDeployment(o, &deployment)

		go func() {
			o.UI.Info(fmt.Sprintf("[%s] starting", d.Config.ID))
			err := d.Run()
			if err != nil {
				o.UI.Error(fmt.Sprintf("[%s] crashed: %s", d.Config.ID, err))
			} else {
				o.UI.Info(fmt.Sprintf("[%s] stopped", d.Config.ID))
			}

			shutdownCh <- struct{}{}
		}()
	}

	for range o.Config.Deployments {
		<-shutdownCh
	}

	return nil
}
