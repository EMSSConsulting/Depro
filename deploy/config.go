package deploy

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Config is the configuration for a deployment agent.
// Some of it can be configured using CLI flags, but most must
// be set using a config file.
type Config struct {
	Server      string        `json:"server"`
	Prefix      string        `json:"prefix"`
	Nodes       int           `json:"nodes"`
	WaitTime    time.Duration `json:"-"`
	WaitTimeRaw string        `json:"wait"`
	AllowStale  bool          `json:"allowStale"`
}

// VersionPath returns the non-/ terminated path for a version key
// such as deploy/myapp/version12345
func (c *Config) VersionPath(version string) string {
	return fmt.Sprintf("%s/%s", strings.Trim(c.Prefix, "/"), strings.Trim(version, "/"))
}

// Merge the second command entry into the first and return a reference
// to the first.
func Merge(a, b *Config) *Config {
	var result = *a

	if b.Server != "" {
		result.Server = b.Server
	}

	if b.Prefix != "" {
		result.Prefix = b.Prefix
	}

	if b.Nodes != 0 {
		result.Nodes = b.Nodes
	}

	if b.WaitTime != 0 {
		result.WaitTime = b.WaitTime
		result.WaitTimeRaw = b.WaitTimeRaw
	}

	if b.AllowStale {
		result.AllowStale = b.AllowStale
	}

	return &result
}

// DefaultConfig returns a pointer to a populated Config object with sensible
// default values.
func DefaultConfig() *Config {
	config := Config{
		Server:      "127.0.0.1:8500",
		Prefix:      "deploy/versions",
		Nodes:       1,
		WaitTime:    10 * time.Minute,
		WaitTimeRaw: "10m",
		AllowStale:  true,
	}

	return &config
}

// ReadConfig reads a configuration file from the given path and returns it.
func ReadConfig(path string) (*Config, error) {
	result := new(Config)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading '%s': %s", path, err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("Error reading '%s': %s", path, err)
	}

	if fi.IsDir() {
		f.Close()
		return nil, fmt.Errorf("Error reading '%s': expected a file, but got a directory instead", path)
	}

	config, err := DecodeConfig(f)
	f.Close()

	if err != nil {
		return nil, fmt.Errorf("Error decoding '%s': %s", path, err)
	}

	result = Merge(result, config)

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
