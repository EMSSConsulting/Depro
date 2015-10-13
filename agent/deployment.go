package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
)

// Deployment describes an individual deployment including the key prefix
// and scripts which should be executed to run the deployment.
type Deployment struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Prefix  string `json:"prefix"`
	Deploy  string `json:"deploy"`
	Rollout string `json:"rollout"`
	Clean   string `json:"clean"`

	state    chan string
	agent    *Operation
	client   *api.Client
	session  *waiter.Session
	versions *Version
}

func (d *Deployment) versionPrefix(version string) string {
	return fmt.Sprintf("%s/%s", strings.Trim(d.Prefix, "/"), strings.Trim(version, "/"))
}

func (d *Deployment) directory(version string) (os.FileInfo, error) {
	f, err := os.Open(d.Path)

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

func (d *Deployment) currentVersion() (string, error) {
	currentVersionFilePath := path.Join(d.Path, "current")

	fContents, err := ioutil.ReadFile(currentVersionFilePath)
	if err != nil {
		return "", err
	}

	return string(fContents), nil
}

func (d *Deployment) updateCurrentVersion(version string) error {
	currentVersionFilePath := path.Join(d.Path, "current")

	return ioutil.WriteFile(currentVersionFilePath, []byte(version), 0)
}

func (d *Deployment) availableVersions() ([]string, error) {
	f, err := os.Open(d.Path)
	if err != nil {
		return nil, err
	}

	fInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !fInfo.IsDir() {
		return nil, fmt.Errorf("Expected deployment path '%s' to be a directory", d.Path)
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
	session, err := waiter.NewSession(d.client, d.ID)
	defer session.Close()

	if err != nil {
		return err
	}

	d.session = session

	return nil
}
