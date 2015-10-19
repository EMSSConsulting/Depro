package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EMSSConsulting/Depro/common"
	"github.com/EMSSConsulting/Depro/util"
)

// Config is the configuration for a deployment agent.
// Some of it can be configured using CLI flags, but most must
// be set using a config file.
type Config struct {
	common.Config

	Name        string             `json:"name"`
	Deployments []DeploymentConfig `json:"deployments"`
}

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

// Merge the second command entry into the first and return a reference
// to the first.
func Merge(a, b *Config) {
	common.Merge(&a.Config, &b.Config)

	a.Deployments = append(a.Deployments, b.Deployments...)
}

// DefaultConfig returns a pointer to a populated Config object with sensible
// default values.
func DefaultConfig() *Config {
	hostname, _ := os.Hostname()

	config := Config{
		Config:      common.DefaultConfig(),
		Name:        hostname,
		Deployments: []DeploymentConfig{},
	}

	return &config
}

// ReadConfig reads a configuration file from the given path and returns it.
func ReadConfig(paths []string) (*Config, error) {
	result := new(Config)

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		if !fi.IsDir() {
			config, err := DecodeConfig(f)
			f.Close()

			if err != nil {
				return nil, fmt.Errorf("Error decoding '%s': %s", path, err)
			}

			Merge(result, config)
			continue
		}

		contents, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		sort.Sort(dirEnts(contents))

		for _, fi := range contents {
			// Don't recursively read contents
			if fi.IsDir() {
				continue
			}

			// If it isn't a JSON file, ignore it
			if !strings.HasSuffix(fi.Name(), ".json") {
				continue
			}

			subpath := filepath.Join(path, fi.Name())
			f, err := os.Open(subpath)
			if err != nil {
				return nil, fmt.Errorf("Error reading '%s': %s", subpath, err)
			}

			config, err := DecodeConfig(f)
			f.Close()

			if err != nil {
				return nil, fmt.Errorf("Error decoding '%s': %s", subpath, err)
			}

			Merge(result, config)
		}
	}

	return result, nil
}

func ParseFlags(config *Config, args []string, flags *flag.FlagSet) error {
	var configFiles []string
	flags.Var((*util.AppendSliceValue)(&configFiles), "config-dir", "directory of json files to read")
	flags.Var((*util.AppendSliceValue)(&configFiles), "config-file", "json file to read config from")

	err := common.ParseFlags(&config.Config, args, flags)
	if err != nil {
		return err
	}

	if len(configFiles) > 0 {
		cFile, err := ReadConfig(configFiles)
		if err != nil {
			return err
		}

		Merge(config, cFile)
	}

	return nil
}

// DecodeConfig decodes a configuration file from an io.Reader stream and returns it.
func DecodeConfig(r io.Reader) (*Config, error) {
	var result Config
	dec := json.NewDecoder(r)

	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	err := result.Finalize()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

type dirEnts []os.FileInfo

func (d dirEnts) Len() int {
	return len(d)
}

func (d dirEnts) Less(i, j int) bool {
	return d[i].Name() < d[j].Name()
}

func (d dirEnts) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
