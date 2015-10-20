package query

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
)

// Operation contains the configuration and clients for performing a deployment
type Operation struct {
	Version string
	UI      cli.Ui
	Config  *Config
}

func NewOperation(ui cli.Ui, config *Config, version string) Operation {
	return Operation{
		Version: version,
		Config:  config,
		UI:      ui,
	}
}

// Run executes the process for a deployment operation
func (o *Operation) Run() error {
	client := o.Config.GetAPIClient()
	kv := client.KV()

	p, _, err := kv.Get(fmt.Sprintf("%s/current", strings.Trim(o.Config.Prefix, "/")), nil)
	if err != nil {
		return err
	}

	if p == nil {
		o.UI.Warn("No version currently rolled out to your cluster, or you specified an incorrect prefix.")
		return nil
	}

	currentVersion := string(p.Value)

	if o.Version == "" {
		o.Version = currentVersion
	}

	if o.Version == currentVersion {
		o.UI.Output(fmt.Sprintf("Version '%s' (active)", o.Version))
	} else {
		o.UI.Output(fmt.Sprintf("Version '%s'", o.Version))
	}

	versionPrefix := fmt.Sprintf("%s/%s", o.Config.Prefix, o.Version)
	ps, _, err := kv.List(versionPrefix, nil)
	if err != nil {
		return err
	}

	for _, p := range ps {
		key := strings.Trim(p.Key[len(versionPrefix):], "/")
		if len(key) > 1 {
			o.UI.Output(fmt.Sprintf("%10s | %s", string(p.Value), key))
		}
	}

	return nil
}
