package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config is the configuration for a deployment agent.
// Some of it can be configured using CLI flags, but most must
// be set using a config file.
type Config struct {
	Name        string        `json:"name"`
	Server      string        `json:"server"`
	WaitTime    time.Duration `json:"-"`
	WaitTimeRaw string        `json:"wait"`
	AllowStale  bool          `json:"allowStale"`

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
func Merge(a, b *Config) *Config {
	var result = *a

	if b.Name != "" {
		result.Name = b.Name
	}

	if b.Server != "" {
		result.Server = b.Server
	}

	if b.WaitTime != 0 {
		result.WaitTime = b.WaitTime
		result.WaitTimeRaw = b.WaitTimeRaw
	}

	if b.AllowStale {
		result.AllowStale = b.AllowStale
	}

	result.Deployments = make([]DeploymentConfig, 0, len(a.Deployments)+len(b.Deployments))
	result.Deployments = append(result.Deployments, a.Deployments...)
	result.Deployments = append(result.Deployments, b.Deployments...)

	return &result
}

// DefaultConfig returns a pointer to a populated Config object with sensible
// default values.
func DefaultConfig() *Config {
	hostname, _ := os.Hostname()

	config := Config{
		Name:        hostname,
		Server:      "127.0.0.1:8500",
		WaitTime:    5 * time.Minute,
		WaitTimeRaw: "5m",
		AllowStale:  true,
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

			result = Merge(result, config)
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

			result = Merge(result, config)
		}
	}

	return result, nil
}

// DecodeConfig decodes a configuration file from an io.Reader stream and returns it.
func DecodeConfig(r io.Reader) (*Config, error) {
	var result Config
	dec := json.NewDecoder(r)

	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	if result.WaitTimeRaw != "" {
		waitTime, err := time.ParseDuration(result.WaitTimeRaw)
		if err != nil {
			return nil, err
		}

		result.WaitTime = waitTime
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
