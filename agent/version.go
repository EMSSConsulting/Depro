package agent

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

type Version struct {
	ID string

	deployment *Deployment
	state      chan string
	client     *api.Client
	customer   *waiter.Customer
	registered bool
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

func (v *Version) deploy() (string, error) {
	v.setState("deploying")
	err := v.recreateDirectory()
	if err != nil {
		v.setState("failed")
		return "", err
	}

	executor := v.getExecutor()

	task := &Task{
		Instructions: v.deployment.Config.Deploy,
	}

	output, err := executor.Run(task)
	if err != nil {
		v.setState("failed")
		return output, err
	}

	v.setState("available")
	return output, nil
}

func (v *Version) rollout() (string, error) {
	v.setState("starting")
	executor := v.getExecutor()

	task := &Task{
		Instructions: v.deployment.Config.Rollout,
	}

	output, err := executor.Run(task)
	if err != nil {
		v.setState("failed")
		return output, err
	}

	v.setState("active")
	return output, nil
}

func (v *Version) clean() (string, error) {
	v.setState("cleaning")
	executor := v.getExecutor()

	task := &Task{
		Instructions: v.deployment.Config.Clean,
	}

	output, err := executor.Run(task)
	if err != nil {
		v.setState("failed")
		return output, err
	}

	err = v.removeDirectory()
	if err != nil {
		v.setState("failed")
		return output, err
	}

	if v.state != nil {
		close(v.state)
	}
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
		shutdownCh := makeShutdownCh()

		select {
		case <-shutdownCh:
			if v.state != nil {
				close(v.state)
			}
			v.state = make(chan string)
		case <-doneCh:
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

func (v *Version) getExecutor() *Executor {
	executor := &Executor{
		Command:   "/usr/bin/sh",
		Arguments: []string{"--login"},
		Extension: ".sh",
		Environment: []string{
			fmt.Sprintf("VERSION='%s'", v.ID),
			fmt.Sprintf("NODE='%s'", v.deployment.agentConfig.Name),
			fmt.Sprintf("DEPLOYMENT='%s'", v.deployment.Config.ID),
		},
		Directory: v.fullPath(),
	}

	switch strings.ToLower(v.deployment.Config.Shell) {
	case "cmd":
		executor.Command = "cmd.exe"
		executor.Arguments = []string{"/Q", "/C"}
		executor.Extension = ".bat"

	case "powershell":
		executor.Command = "powershell.exe"
		executor.Arguments = []string{"-noprofile", "-noninteractive", "-executionpolicy", "Bypass", "-command"}
		executor.Extension = ".ps1"
	}

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

// makeShutdownCh creates a channel which will emit whenever a SIGTERM/SIGINT
// is received by the application - this is used to close any active sessions.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}
