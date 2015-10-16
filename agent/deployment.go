package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/EMSSConsulting/Depro/util"
	"github.com/EMSSConsulting/waiter"
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

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

	deployVersion   chan string
	rolloutVersion  chan string
	cleanVersion    chan string
	registerVersion chan string
}

func NewDeployment(operation *Operation, config *DeploymentConfig) *Deployment {
	d := &Deployment{
		Config: config,

		agentConfig: operation.Config,
		client:      operation.Config.GetAPIClient(),
		ui:          operation.UI,
		versions:    map[string]*Version{},

		deployVersion:   make(chan string),
		rolloutVersion:  make(chan string),
		cleanVersion:    make(chan string),
		registerVersion: make(chan string),
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

func (d *Deployment) fetchCurrentVersion(waitIndex uint64) (string, uint64, error) {
	kv := d.client.KV()

	key, meta, err := kv.Get(fmt.Sprintf("%s/current", d.Config.Prefix), &api.QueryOptions{
		WaitIndex: waitIndex,
	})

	if err != nil {
		return "", 0, err
	}

	if key == nil {
		return "", meta.LastIndex, nil
	}

	return string(key.Value), meta.LastIndex, nil
}

func (d *Deployment) diffVersions(oldVersions, newVersions []string) {
	oldVersionsSet := util.SliceToMap(oldVersions)
	newVersionsSet := util.SliceToMap(newVersions)

	for _, id := range oldVersions {
		_, exists := newVersionsSet[id]
		if !exists {
			d.cleanVersion <- id
		}
	}

	for _, id := range newVersions {
		_, exists := oldVersionsSet[id]
		if !exists {
			d.deployVersion <- id
		}
	}
}

func (d *Deployment) diffCurrentVersion(oldVersion, newVersion string) {
	if oldVersion != newVersion {
		if newVersion != "" {
			d.rolloutVersion <- newVersion
		}
	}
}

func (d *Deployment) watchVersions() error {
	versions := []string{}

	lastWaitIndex := uint64(0)

	for range util.NotShutdown() {
		newVersions, nextWaitIndex, err := d.fetchVersions(lastWaitIndex)
		if err != nil {
			return err
		}

		d.diffVersions(versions, newVersions)

		versions = newVersions
		lastWaitIndex = nextWaitIndex
	}

	return nil
}

func (d *Deployment) watchCurrentVersion() error {
	currentVersion := d.currentVersion()

	lastWaitIndex := uint64(0)

	for range util.NotShutdown() {
		newCurrentVersion, newWaitIndex, err := d.fetchCurrentVersion(lastWaitIndex)
		if err != nil {
			return err
		}

		lastWaitIndex = newWaitIndex

		d.diffCurrentVersion(currentVersion, newCurrentVersion)
		currentVersion = newCurrentVersion
	}

	return nil
}

func (d *Deployment) Run() error {
	session, err := waiter.NewSession(d.client, d.Config.ID)
	defer session.Close()

	if err != nil {
		return err
	}

	d.session = session

	go func() {
		for id := range d.registerVersion {
			version, exists := d.versions[id]

			if !exists {
				continue
			}

			go func(version *Version) {
				err := version.register()
				if err != nil {
					d.ui.Error(fmt.Sprintf("[%s] version '%s' not registered: %s", d.Config.ID, version.ID, err))
				}
			}(version)
		}
	}()

	go func() {
		for id := range d.deployVersion {
			version, exists := d.versions[id]

			if !exists {
				version = newVersion(d, id)
				d.versions[id] = version
				d.registerVersion <- id
			}

			output, err := version.deploy()
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' deployment failed: %s", d.Config.ID, version.ID, err))
			} else {
				d.ui.Output(fmt.Sprintf("[%s] version '%s' deployed", d.Config.ID, id))
			}
			d.ui.Info(output)
		}
	}()

	go func() {
		for id := range d.rolloutVersion {
			version, exists := d.versions[id]
			if !exists {
				version = newVersion(d, id)
				d.versions[id] = version

				if !version.exists() {
					version.close <- struct{}{}
					d.ui.Error(fmt.Sprintf("[%s] version '%s' not found", d.Config.ID, id))
					continue
				} else {
					d.registerVersion <- id
					version.state <- "available"
				}
			}

			output, err := version.rollout()
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' rollout failed: %s", d.Config.ID, version.ID, err))
			}

			d.ui.Info(output)

			err = d.updateCurrentVersion(id)
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' rollout not persisted: %s", d.Config.ID, version.ID, err))
			}
		}
	}()

	go func() {
		for id := range d.cleanVersion {
			version, exists := d.versions[id]

			if !exists {
				continue
			}

			output, err := version.clean()
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' cleanup failed: %s", d.Config.ID, version.ID, err))
			} else {
				d.ui.Output(fmt.Sprintf("[%s] version '%s' removed", d.Config.ID, id))
			}
			d.ui.Info(output)
		}
	}()

	doneCh := make(chan struct{}, 2)

	go func() {
		err := d.watchCurrentVersion()
		if err != nil {
			d.ui.Error(fmt.Sprintf("[%s] crashed: %s", d.Config.ID, err))
		}
		doneCh <- struct{}{}
	}()

	go func() {
		err := d.watchVersions()
		if err != nil {
			d.ui.Error(fmt.Sprintf("[%s] crashed: %s", d.Config.ID, err))
		}
		doneCh <- struct{}{}
	}()

	go func() {
		select {
		case <-util.MakeShutdownCh():
			close(d.deployVersion)
			close(d.rolloutVersion)
			close(d.cleanVersion)
			close(d.registerVersion)
		}
	}()

	<-doneCh
	<-doneCh
	return nil
}
