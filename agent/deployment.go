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
	Shell   string   `json:"shell"`
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
	versions    map[string]*Version
}

func NewDeployment(operation *Operation, config *DeploymentConfig) *Deployment {
	d := &Deployment{
		Config: config,

		agentConfig: operation.Config,
		client:      operation.Client,
		ui:          operation.UI,
		versions:    map[string]*Version{},
	}

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

func (d *Deployment) fetchVersions(waitIndex uint64) ([]string, uint64, error) {
	kv := d.client.KV()

	keys, meta, err := kv.Keys(fmt.Sprintf("%s/", d.Config.Prefix), "/", &api.QueryOptions{
		WaitIndex: waitIndex,
	})

	if err != nil {
		return nil, 0, err
	}

	fixedKeys := keys[:0]
	for _, key := range keys {
		fixedKey := strings.Trim(key[len(d.Config.Prefix):], "/")
		if fixedKey == "" {
			continue
		}

		if fixedKey == "current" {
			continue
		}

		fixedKeys = append(fixedKeys, fixedKey)
	}

	return fixedKeys, meta.LastIndex, nil
}

func (d *Deployment) watchVersions() error {
	versions, err := d.availableVersions()
	if err != nil {
		return err
	}

	lastWaitIndex := uint64(0)

	for _, version := range versions {
		d.versions[version] = newVersion(d, version)
	}

	done := false

	go func() {
		select {
		case <-makeShutdownCh():
			done = true
			d.ui.Info(fmt.Sprintf("[%s] shutting down", d.Config.ID))
		}
	}()

	for !done {
		newVersions, nextWaitIndex, err := d.fetchVersions(lastWaitIndex)
		if err != nil {
			return err
		}

		newVersionSet := map[string]struct{}{}
		for _, version := range newVersions {
			newVersionSet[version] = struct{}{}

			_, exists := d.versions[version]

			if !exists {
				d.ui.Output(fmt.Sprintf("[%s] found '%s'", d.Config.ID, version))
				d.versions[version] = newVersion(d, version)
			}

			v := d.versions[version]
			if !v.registered {
				go func() {
					err := v.register()
					if err != nil {
						d.ui.Error(fmt.Sprintf("[%s] failed '%s': %s", d.Config.ID, version, err))
					}
				}()
			}

			if !exists {
				go func(version *Version) {
					output, err := version.deploy()
					if err != nil {
						d.ui.Error(fmt.Sprintf("[%s] version '%s' deployment failed: %s", d.Config.ID, version.ID, err))
					}
					d.ui.Info(output)
				}(v)
			}
		}

		for id, version := range d.versions {
			_, exists := newVersionSet[id]
			if !exists {
				d.ui.Output(fmt.Sprintf("[%s] removed '%s'", d.Config.ID, id))
				go func(id string, version *Version) {
					output, err := version.clean()
					if err != nil {
						d.ui.Error(fmt.Sprintf("[%s]@%s cleanup failed: %s", d.Config.ID, version.ID, err))
					}
					d.ui.Info(output)
				}(id, version)
			}
		}

		lastWaitIndex = nextWaitIndex

	}

	return nil
}

func (d *Deployment) watchCurrentVersion() error {
	return nil
}

func (d *Deployment) Run() error {
	session, err := waiter.NewSession(d.client, d.Config.ID)
	defer session.Close()

	if err != nil {
		return err
	}

	d.session = session

	return d.watchVersions()
}
