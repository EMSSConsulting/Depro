package deploy

import (
	"fmt"
	"time"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// Operation contains the configuration and clients for performing a deployment
type Operation struct {
	Version string
	UI      cli.Ui
	Config  *Config
	Client  *api.Client

	wait *waiter.Wait
}

// Run executes the process for a deployment operation
func (o *Operation) Run() error {
	o.wait = waiter.NewWaiter(
		o.Client,
		o.Config.VersionPath(o.Version),
		o.Config.Nodes,
		func(w *waiter.WaitNode) bool {
			return w.State == "ready"
		})

	errorCh := make(chan error)

	go func() {
		allReady, err := o.wait.Wait(o.Config.WaitTime)

		if !allReady && err == nil {
			err = fmt.Errorf("Deployment failed or timed out during preperation phase.")
		}

		time.Sleep(1 * time.Millisecond) // UI tweak to ensure that AllReady() is triggered before completion

		errorCh <- err
	}()

	for {
		select {
		case node := <-o.wait.NodeUpdate():
			o.UI.Info(fmt.Sprintf("Node '%s' is now '%s' (from '%s')", node.Node, node.State, node.LastState))
		case <-o.wait.AllReady():
			o.UI.Info(fmt.Sprintf("Version '%s' deployed to all nodes, ready for rollout.", o.Version))
		case err := <-errorCh:
			return err
		}
	}
}
