package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/executor"
	"github.com/EMSSConsulting/Depro/util"
	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

type Version struct {
	ID string

	deployment *Deployment
	state      chan string
	close      chan struct{}
	client     *api.Client
	customer   *waiter.Customer
	registered bool
}

func newVersion(deployment *Deployment, id string) *Version {
	v := &Version{
		ID:         id,
		deployment: deployment,
		client:     deployment.client,
		state:      make(chan string),
		close:      make(chan struct{}),
	}

	v.customer = waiter.NewCustomer(v.client, deployment.versionPrefix(id), deployment.agentConfig.Name, v.state)

	go func() {
		select {
		case <-v.close:
			close(v.state)
			delete(v.deployment.versions, v.ID)
			v.state = make(chan string)
		}
	}()

	return v
}

func (v *Version) deploy() (string, error) {
	v.setState("deploying")
	output := fmt.Sprintf("Preparing directory '%s'\n", v.fullPath())

	err := v.recreateDirectory()
	if err != nil {
		v.state <- "failed"
		return "", err
	}

	if len(v.deployment.Config.Deploy) > 0 {
		ex := v.getExecutor()

		task, err := executor.NewTask(v.deployment.Config.Deploy, nil, nil)
		if err != nil {
			v.state <- "failed"
			return output, err
		}

		cmdOutput, err := ex.RunOutput(task)
		output = output + string(cmdOutput)
		if err != nil {
			v.state <- "failed"
			return output, err
		}
	}

	v.state <- "available"
	return output, nil
}

func (v *Version) rollout() (string, error) {
	output := ""

	v.setState("starting")
	ex := v.getExecutor()

	task, err := executor.NewTask(v.deployment.Config.Rollout, nil, nil)
	if err != nil {
		v.state <- "failed"
		return output, err
	}

	cmdOutput, err := ex.RunOutput(task)
	output = output + string(cmdOutput)
	if err != nil {
		v.state <- "failed"
		return output, err
	}

	v.state <- "active"
	return output, nil
}

func (v *Version) clean() (string, error) {
	output := ""

	if len(v.deployment.Config.Clean) > 0 {
		ex := v.getExecutor()

		task, err := executor.NewTask(v.deployment.Config.Clean, nil, nil)
		if err != nil {
			v.state <- "failed"
			return output, err
		}

		cmdOutput, err := ex.RunOutput(task)
		output = output + string(cmdOutput)
	}

	err := v.removeDirectory()
	if err != nil {
		return output, err
	}

	v.close <- struct{}{}

	return output, nil
}

// register publishes an entry in the correct version node on the server
// to inform watchers of the state of the local copy of this version.
func (v *Version) register() error {
	v.registered = true
	defer func() { v.registered = false }()

	doneCh := make(chan struct{})

	// Function to shutdown this version's goroutines when the application
	// requests an exit.
	go func() {
		shutdownCh := util.MakeShutdownCh()

		select {
		case <-shutdownCh:
			v.close <- struct{}{}
		case <-doneCh:
			v.close <- struct{}{}
		}
	}()

	err := v.customer.Run(v.deployment.session)

	doneCh <- struct{}{}
	return err
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

func (v *Version) getExecutor() executor.Executor {
	executor := executor.NewExecutor(strings.ToLower(v.deployment.Config.Shell))

	executor.Environment["VERSION"] = v.ID
	executor.Environment["NODE"] = v.deployment.agentConfig.Name
	executor.Environment["DEPLOYMENT"] = v.deployment.Config.ID
	executor.Directory = v.fullPath()

	return executor
}

func (v *Version) directory() (os.FileInfo, error) {
	return v.deployment.directory(v.ID)
}

func (v *Version) fullPath() string {
	return v.deployment.fullPath(v.ID)
}

func (v *Version) recreateDirectory() error {
	v.removeDirectory()

	return os.MkdirAll(v.fullPath(), os.ModeDir)
}

func (v *Version) removeDirectory() error {
	if v.exists() {
		err := os.RemoveAll(v.fullPath())
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Version) exists() bool {
	_, err := v.directory()
	return err == nil || os.IsNotExist(err)
}
