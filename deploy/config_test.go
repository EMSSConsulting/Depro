package deploy

import (
	"bytes"
	"testing"
	"time"

	"github.com/EMSSConsulting/Depro/common"
)

func TestDecodeConfig_Server(t *testing.T) {
	input := `{"server": "127.0.0.1:8500"}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if config.Server != "127.0.0.1:8500" {
		t.Fatalf("bad server, got '%s', expected '%s'", config.Server, "127.0.0.1:8500")
	}
}

func TestDecodeConfig_Prefix(t *testing.T) {
	input := `{"prefix": "myapp/production/versions"}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if config.Prefix != "myapp/production/versions" {
		t.Fatalf("bad prefix, got '%s', expected '%s'", config.Prefix, "myapp/production/versions")
	}
}

func TestDecodeConfig_Nodes(t *testing.T) {
	input := `{"nodes": 3}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if config.Nodes != 3 {
		t.Fatalf("bad nodes, got '%d', expected '%d'", config.Nodes, 3)
	}
}

func TestDecodeConfig_WaitTime(t *testing.T) {
	input := `{"wait": "10s"}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if config.WaitTime != 10*time.Second {
		t.Fatalf("bad wait time, got '%v', expected '%v'", config.WaitTime, 10*time.Second)
	}
}

func TestDecodeConfig_AllowStale(t *testing.T) {
	input := `{"allowStale": true}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !config.AllowStale {
		t.Fatalf("bad allowStale, got '%v', expected '%v'", config.AllowStale, true)
	}
}

func TestMerge(t *testing.T) {
	c1 := Config{
		Config: common.Config{
			Server: "127.0.0.1:8000",
		},
	}

	c2 := Config{
		Config: common.Config{
			Prefix: "test",
		},
	}

	Merge(&c1, &c2)

	if c1.Server != "127.0.0.1:8000" {
		t.Fatal("bad server field")
	}

	if c1.Prefix != "test" {
		t.Fatal("bad server field")
	}
}

func TestVersionPath(t *testing.T) {
	conf := Config{
		Config: common.Config{
			Prefix: "myapp/test/version/",
		},
	}

	if conf.VersionPath("1234") != "myapp/test/version/1234" {
		t.Fatalf("Version path not generated correctly, got '%s' but expected '%s'", conf.VersionPath("1234"), "myapp/test/version/1234")
	}

	if conf.VersionPath("1234/") != "myapp/test/version/1234" {
		t.Fatalf("Version path not generated correctly, got '%s' but expected '%s'", conf.VersionPath("1234"), "myapp/test/version/1234")
	}
}
