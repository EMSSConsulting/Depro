package agent

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/util"
	"github.com/EMSSConsulting/Executor"
	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

type Version struct {
	ID string

	deployment *Deployment
	state      chan string
	lastState  string
	close      chan struct{}
	client     *api.Client
	customer   *waiter.Customer
	registered bool
	log        *log.Logger
}

func newVersion(deployment *Deployment, id string) *Version {
	v := &Version{
		ID:         id,
		deployment: deployment,
		client:     deployment.client,
		state:      make(chan string),
		lastState:  "unregistered",
		close:      make(chan struct{}),
		log:        log.New(os.Stdout, fmt.Sprintf("[%s@%s]", deployment.Config.ID, id), log.Ltime),
	}

	v.customer = waiter.NewCustomer(v.client, deployment.versionPrefix(id), deployment.agentConfig.Name, v.state)

	return v
}

func (v *Version) deploy() (string, error) {
	v.setState("deploying", true)
	output := fmt.Sprintf("Preparing directory '%s'\n", v.fullPath())

	err := v.recreateDirectory()
	if err != nil {
		v.setState("failed", false)
		return "", err
	}

	if len(v.deployment.Config.Deploy) > 0 {
		ex := v.getExecutor()

		task, err := executor.NewTask(v.deployment.Config.Deploy, nil, nil)
		if err != nil {
			v.setState("failed", false)
			return output, err
		}

		cmdOutput, err := ex.RunOutput(task)
		output = output + string(cmdOutput)
		if err != nil {
			v.setState("failed", false)
			return output, err
		}
	}

	v.setState("available", false)
	return output, nil
}

func (v *Version) rollout() (string, error) {
	output := ""

	v.setState("starting", true)
	ex := v.getExecutor()

	task, err := executor.NewTask(v.deployment.Config.Rollout, nil, nil)
	if err != nil {
		v.setState("failed", false)
		return output, err
	}

	cmdOutput, err := ex.RunOutput(task)
	output = output + string(cmdOutput)
	if err != nil {
		v.setState("failed", false)
		return output, err
	}

	v.setState("active", false)
	return output, nil
}

func (v *Version) clean() (string, error) {
	output := ""

	if len(v.deployment.Config.Clean) > 0 {
		ex := v.getExecutor()

		task, err := executor.NewTask(v.deployment.Config.Clean, nil, nil)
		if err != nil {
			v.setState("failed", false)
			return output, err
		}

		cmdOutput, err := ex.RunOutput(task)
		output = output + string(cmdOutput)
	}

	err := v.removeDirectory()
	if err != nil {
		return output, err
	}

	v.shutdown()

	return output, nil
}

// register publishes an entry in the correct version node on the server
// to inform watchers of the state of the local copy of this version.
func (v *Version) register() error {
	v.log.Printf("registering\n")
	v.registered = true
	defer func() { v.registered = false }()

	doneCh := make(chan struct{})

	// Function to shutdown this version's goroutines when the application
	// requests an exit.
	go func() {
		shutdownCh := util.MakeShutdownCh()

		select {
		case <-shutdownCh:
			v.shutdown()
		case <-doneCh:
			v.shutdown()
		}
	}()

	err := v.customer.Run(v.deployment.session)

	if err != nil {
		v.log.Printf("registration failed: %s", err)
	} else {
		v.log.Printf("deregistered")
	}

	close(doneCh)
	return err
}

func (v *Version) shutdown() {
	if v.state == nil {
		return
	}

	v.log.Printf("shutting down\n")

	close(v.state)
	close(v.close)
	delete(v.deployment.versions, v.ID)

	v.state = nil
}

// setState sets the state of this version entry in a non-blocking manner.
// It should only be called once v.register() has been started in a goroutine.
// Failure to do so will result in your state change being lost.
func (v *Version) setState(state string, async bool) {
	v.log.Printf("{%s}\n", state)

	if async {
		select {
		case v.state <- state:
		default:
		}
	} else {
		v.state <- state
	}

	v.lastState = state
}

func (v *Version) getExecutor() executor.Executor {
	executor := executor.NewExecutor(strings.ToLower(v.deployment.Config.Shell))

	executor.Environment["VERSION"] = v.ID
	executor.Environment["AGENT_NAME"] = v.deployment.agentConfig.Name
	executor.Environment["DEPLOYMENT_ID"] = v.deployment.Config.ID
	executor.Environment["DEPLOYMENT_PREFIX"] = v.deployment.Config.Prefix
	executor.Environment["DEPLOYMENT_PATH"] = v.deployment.Config.Path
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

	return os.MkdirAll(v.fullPath(), os.ModeDir|os.ModePerm)
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
	f, err := os.Open(v.deployment.fullPath(v.ID))

	if err != nil {
		return false
	}

	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return false
	}

	if !fInfo.IsDir() {
		return false
	}

	return true
}
