package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// Deployment describes an individual deployment including the key prefix
// and scripts which should be executed to run the deployment.
type DeploymentConfig struct {
	ID      string   `json:"id"`
	Path    string   `json:"path"`
	Prefix  string   `json:"prefix"`
	Deploy  []string `json:"deploy"`
	Rollout []string `json:"rollout"`
	Clean   []string `json:"clean"`
}

// Deployment describes the internal state of a deployment which consists
// of multiple versions.
// Deployments are each responsible for controlling the lifespan of their
// session as they may be created and destroyed at any time.
type Deployment struct {
	Config *DeploymentConfig

	agentConfig *Config
	client      *api.Client
	ui          cli.Ui
	session     *waiter.Session
	versions    []Version
}

func NewDeployment(operation *Operation, config *DeploymentConfig) *Deployment {
	d := &Deployment{
		Config: config,

		agentConfig: operation.Config,
		client:      operation.Client,
		ui:          operation.UI,
	}

	d.versions = make([]Version, 0)

	return d
}

func (d *Deployment) versionPrefix(version string) string {
	return fmt.Sprintf("%s/%s", strings.Trim(d.Config.Prefix, "/"), strings.Trim(version, "/"))
}

func (d *Deployment) fullPath(version string) string {
	if version == "" {
		return d.Config.Path
	}

	return path.Join(d.Config.Path, version)
}

func (d *Deployment) directory(version string) (os.FileInfo, error) {
	f, err := os.Open(d.Config.Path)

	defer f.Close()

	if err != nil {
		return nil, err
	}

	fInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if version == "" {
		return fInfo, err
	}

	contents, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, content := range contents {
		if content.IsDir() && content.Name() == version {
			return content, nil
		}
	}

	return nil, fmt.Errorf("Could not find a deployment directory called '%s'", version)
}

func (d *Deployment) currentVersion() string {
	currentVersionFilePath := path.Join(d.Config.Path, "current")

	fContents, err := ioutil.ReadFile(currentVersionFilePath)
	if err != nil {
		return ""
	}

	return string(fContents)
}

func (d *Deployment) updateCurrentVersion(version string) error {
	currentVersionFilePath := path.Join(d.Config.Path, "current")

	return ioutil.WriteFile(currentVersionFilePath, []byte(version), 0)
}

func (d *Deployment) availableVersions() ([]string, error) {
	f, err := os.Open(d.Config.Path)
	if err != nil {
		return nil, err
	}

	fInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !fInfo.IsDir() {
		return nil, fmt.Errorf("Expected deployment path '%s' to be a directory", d.Config.Path)
	}

	children, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	versions := []string{}
	for _, child := range children {
		if child.IsDir() {
			versions = append(versions, child.Name())
		}
	}

	return versions, nil
}

func (d *Deployment) Run() error {
	session, err := waiter.NewSession(d.client, d.Config.ID)
	defer session.Close()

	if err != nil {
		return err
	}

	d.session = session

	return nil
}
