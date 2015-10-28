package agent

import (
	"fmt"
	"io/ioutil"
	"log"
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

	agentConfig    *Config
	client         *api.Client
	ui             cli.Ui
	session        *waiter.Session
	versions       map[string]*Version

	log *log.Logger
	err *log.Logger

	deployVersion   chan *Version
	rolloutVersion  chan *Version
	cleanVersion    chan *Version
	registerVersion chan *Version
}

func NewDeployment(operation *Operation, config *DeploymentConfig) *Deployment {
	d := &Deployment{
		Config: config,

		agentConfig: operation.Config,
		client:      operation.Config.GetAPIClient(),
		ui:          operation.UI,
		versions:    map[string]*Version{},

		log: log.New(os.Stdout, fmt.Sprintf("[%s]", config.ID), log.Ltime),
		err: log.New(os.Stderr, fmt.Sprintf("ERROR: [%s]", config.ID), log.Ltime|log.Lshortfile),

		deployVersion:   make(chan *Version),
		rolloutVersion:  make(chan *Version),
		cleanVersion:    make(chan *Version),
		registerVersion: make(chan *Version),
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

	if err != nil {
		return nil, err
	}

	defer f.Close()

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

	panic(fmt.Errorf("deployment directory for {%s} not found\n", version))
	d.err.Printf("deployment directory for {%s} not found\n", version)

	return nil, fmt.Errorf("Could not find a deployment directory called '%s'", version)
}

func (d *Deployment) currentVersion() string {
	currentVersionFilePath := path.Join(d.Config.Path, "current")

	fContents, err := ioutil.ReadFile(currentVersionFilePath)
	if err != nil {
		d.log.Printf("no current version file present\n")
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
		d.err.Printf("deployment path {%s} was not a directory\n", d.Config.Path)
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
		if !exists && d.cleanVersion != nil {
			d.log.Printf("got remove {%s}\n", id)

			version, exists := d.versions[id]
			if exists {
				d.cleanVersion <- version
			}
		}
	}

	for _, id := range newVersions {
		_, exists := oldVersionsSet[id]
		if !exists && d.deployVersion != nil {
			d.log.Printf("got deploy {%s}\n", id)
			version, exists := d.versions[id]
			if !exists {
				version = newVersion(d, id)
				d.versions[id] = version

				d.registerVersion <- version
			}

			if version.exists() {
				if version.ID == d.currentVersion() {
					d.rolloutVersion <- version
				} else {
					version.setState("available", false)
				}
			} else {
				d.deployVersion <- version
			}
		}
	}
}

func (d *Deployment) diffCurrentVersion(oldVer, newVer string) {
	if oldVer != newVer {
		if newVer != "" && d.rolloutVersion != nil {
			d.log.Printf("got rollout {%s}\n", newVer)

			version, exists := d.versions[newVer]
			if !exists {
				version = newVersion(d, newVer)
				d.versions[newVer] = version

				d.registerVersion <- version
			}

			if !version.exists() {
				d.deployVersion <- version
			} else {
				d.rolloutVersion <- version
			}
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

	if err != nil {
		return err
	}

	defer session.Close()

	d.session = session

	go func() {
		for version := range d.registerVersion {
			go func() {
				err := version.register()
				if err != nil {
					d.err.Printf("could not register {%s}: %s\n", version.ID, err)
					d.ui.Error(fmt.Sprintf("[%s] version '%s' not registered: %s", d.Config.ID, version.ID, err))
				}
			}()
		}
	}()

	go func() {
		for version := range d.deployVersion {
			if !version.exists() {
				output, err := version.deploy()
				if err != nil {
					d.ui.Error(fmt.Sprintf("[%s] version '%s' deployment failed: %s", d.Config.ID, version.ID, err))
				} else {
					d.ui.Output(fmt.Sprintf("[%s] version '%s' deployed", d.Config.ID, version.ID))
				}
				d.ui.Info(output)
			}

			// Rollout this version since it has only been deployed now
			if version.ID == d.currentVersion() {
				d.rolloutVersion <- version
			}
		}
	}()

	go func() {
		for version := range d.rolloutVersion {
			output, err := version.rollout()
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' rollout failed: %s", d.Config.ID, version.ID, err))
			}

			d.ui.Info(output)

			for _, otherVersion := range d.versions {
				if otherVersion != version && otherVersion.exists() {
					otherVersion.setState("available", true)
				}
			}
		}
	}()

	go func() {
		for version := range d.cleanVersion {
			output, err := version.clean()
			if err != nil {
				d.ui.Error(fmt.Sprintf("[%s] version '%s' cleanup failed: %s", d.Config.ID, version.ID, err))
			} else {
				d.ui.Output(fmt.Sprintf("[%s] version '%s' removed", d.Config.ID, version.ID))
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
