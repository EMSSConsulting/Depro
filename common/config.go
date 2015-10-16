package common

import "time"

// Config is the configuration for a deployment agent.
// Some of it can be configured using CLI flags, but most must
// be set using a config file.
type Config struct {
	Server      string        `json:"server"`
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	Datacenter  string        `json:"datacenter"`
	Prefix      string        `json:"prefix"`
	WaitTime    time.Duration `json:"-"`
	WaitTimeRaw string        `json:"wait"`
	AllowStale  bool          `json:"allowStale"`
}

// DefaultConfig returns a pointer to a populated Config object with sensible
// default values.
func DefaultConfig() Config {
	return Config{
		Server:      "127.0.0.1:8500",
		Username:    "",
		Password:    "",
		Datacenter:  "",
		Prefix:      "deploy/versions",
		WaitTime:    10 * time.Minute,
		WaitTimeRaw: "10m",
		AllowStale:  true,
	}
}

// Merge the second command entry into the first and return a reference
// to the first.
func Merge(a, b *Config) {
	if b.Server != "" {
		a.Server = b.Server
	}

	if b.Datacenter != "" {
		a.Datacenter = b.Datacenter
	}

	if b.Prefix != "" {
		a.Prefix = b.Prefix
	}

	if b.WaitTime != 0 {
		a.WaitTime = b.WaitTime
		a.WaitTimeRaw = b.WaitTimeRaw
	}

	if b.Username != "" {
		a.Username = b.Username
	}

	if b.Password != "" {
		a.Password = b.Password
	}

	if b.AllowStale {
		a.AllowStale = b.AllowStale
	}
}

// Finalize is responsible for performing any final conversions, such as
// timeouts.
func (c *Config) Finalize() error {
	if c.WaitTimeRaw != "" {
		waitTime, err := time.ParseDuration(c.WaitTimeRaw)
		if err != nil {
			return err
		}

		c.WaitTime = waitTime
	}

	return nil
}
