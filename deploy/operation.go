package deploy

import (
	"fmt"

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
			switch w.State {
			case "ready":
				fallthrough
			case "available":
				fallthrough
			case "failed":
				fallthrough
			case "active":
				return true

			}

			return false
		})

	errorCh := make(chan error)

	go func() {
		o.UI.Info(fmt.Sprintf("Starting deployment of version '%s'", o.Version))
		allReady, err := o.wait.Wait(o.Config.WaitTime)

		if !allReady && err == nil {
			err = fmt.Errorf("Deployment failed or timed out during preperation phase.")
		}

		if err != nil {
			select {
			case errorCh <- err:
			default:
			}
		}
	}()

	for {
		select {
		case node := <-o.wait.NodeUpdate:
			if node.State == "" && node.LastState == "" {
				o.UI.Info(fmt.Sprintf("+ %s", node.Node))
			} else if node.State == "" {
				o.UI.Info(fmt.Sprintf("- %s #%s", node.Node, node.LastState))
			} else if node.LastState == "" {
				o.UI.Info(fmt.Sprintf("+ %s #%s", node.Node, node.State))
			} else {
				o.UI.Info(fmt.Sprintf("> %s #%s -> #%s", node.Node, node.LastState, node.State))
			}
		case node := <-o.wait.NodeReady:
			o.UI.Output(fmt.Sprintf("+ %s@%s", o.Version, node.Node))
		case nodes := <-o.wait.AllReady:
			successful := true
			for _, node := range nodes {
				if node.State == "failed" {
					o.UI.Warn(fmt.Sprintf("! %s #failed", node.Node))
					successful = false
				}
			}
			if successful {
				o.UI.Info(fmt.Sprintf("Version '%s' deployed to all nodes, ready for rollout.", o.Version))
			} else {
				return fmt.Errorf("Version '%s' deployment failed", o.Version)
			}

			return nil
		case err := <-errorCh:
			return err
		}
	}
}
