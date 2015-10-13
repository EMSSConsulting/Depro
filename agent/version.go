package agent

import (
	"os"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

type Version struct {
	ID string

	deployment *Deployment
	state      chan string
	client     *api.Client
	customer   *waiter.Customer
}

func newVersion(deployment *Deployment, id string) *Version {
	v := &Version{
		ID:         id,
		deployment: deployment,
		client:     deployment.client,
	}

	v.customer = waiter.NewCustomer(v.client, deployment.versionPrefix(id), deployment.agentConfig.Name, v.state)

	return v
}

func (v *Version) deploy() error {
	v.setState("busy")
	err := v.recreateDirectory()
	if err != nil {
		return err
	}
	
	v.setState("ready")
	return nil
}

func (v *Version) rollout() error {
	return nil
}

func (v *Version) clean() error {
	return nil
}

// register publishes an entry in the correct version node on the server
// to inform watchers of the state of the local copy of this version.
func (v *Version) register() error {
	return v.customer.Run(v.deployment.session)
}

// setState sets the state of this version entry in a non-blocking manner.
// It should only be called once v.register() has been started in a goroutine.
// Failure to do so will result in your state change being lost. 
func (v *Version) setState(state string) {
	select {
		case v.state <- state:
		default:
	}
}

func (v *Version) directory() (os.FileInfo, error) {
	return v.deployment.directory(v.ID)
}

func (v *Version) fullPath() string {
	return v.deployment.fullPath(v.ID)
}

func (v *Version) recreateDirectory() error {
	if v.exists() {
		err := os.RemoveAll(v.fullPath())
		if err != nil {
			return err
		}
	}
	
	return os.MkdirAll(v.fullPath(), os.ModeDir)
}

func (v *Version) exists() bool {
	_, err := v.directory()
	return err == nil || os.IsNotExist(err)
}
