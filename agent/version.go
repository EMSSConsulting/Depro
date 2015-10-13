package agent

import (
	"fmt"
	"os"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

type Version struct {
	ID         string
	Deployment *Deployment

	state    chan string
	client   *api.Client
	customer *waiter.Customer
}

func newVersion(deployment *Deployment, id string) *Version {
	v := &Version{
		ID:         id,
		Deployment: deployment,
		client:     deployment.client,
	}

	v.customer = waiter.NewCustomer(deployment.client, deployment.versionPrefix(id), deployment.agent.Config.Name, v.state)

	go func(deployment *Deployment, id string) {
		err := v.customer.Run(deployment.session)
		if err != nil {
			deployment.agent.UI.Error(fmt.Sprintf("Failed to create entry for '%s@%s'", deployment.ID, id))
		}
	}(deployment, id)

	return v
}

func (v *Version) deploy() error {
	return nil
}

func (v *Version) directory() (os.FileInfo, error) {
	return v.Deployment.directory(v.ID)
}

func (v *Version) exists() bool {
	_, err := v.directory()
	return err == nil || os.IsNotExist(err)
}
