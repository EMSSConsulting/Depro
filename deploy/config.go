package deploy

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/EMSSConsulting/Depro/common"
)

// Config is the configuration for a deployment agent.
// Some of it can be configured using CLI flags, but most must
// be set using a config file.
type Config struct {
	common.Config

	Nodes int `json:"nodes"`
}

// VersionPath returns the non-/ terminated path for a version key
// such as deploy/myapp/version12345
func (c *Config) VersionPath(version string) string {
	return fmt.Sprintf("%s/%s", strings.Trim(c.Prefix, "/"), strings.Trim(version, "/"))
}

// DefaultConfig returns a pointer to a populated Config object with sensible
// default values.
func DefaultConfig() *Config {
	config := Config{
		Config: common.DefaultConfig(),
		Nodes:  1,
	}

	return &config
}

// Merge the second command entry into the first and return a reference
// to the first.
func Merge(a, b *Config) {
	common.Merge(&a.Config, &b.Config)

	if b.Nodes != 0 {
		a.Nodes = b.Nodes
	}
}

func ParseFlags(config *Config, args []string, flags *flag.FlagSet) error {

	var configFile string
	flags.StringVar(&configFile, "config", "", "")

	flags.IntVar(&config.Nodes, "nodes", 1, "minimum number of nodes to deploy to")

	err := common.ParseFlags(&config.Config, args, flags)
	if err != nil {
		return err
	}

	if configFile != "" {
		cFile, err := ReadConfig(configFile)
		if err != nil {
			return err
		}

		Merge(config, cFile)
	}

	return nil
}

// ReadConfig reads a configuration file from the given path and returns it.
func ReadConfig(path string) (*Config, error) {
	result := DefaultConfig()

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

	Merge(result, config)

	return result, nil
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
