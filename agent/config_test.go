package agent

import (
	"bytes"
	"testing"
	"time"
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
	input := `{"deployments": [{ "prefix": "myapp/production/versions"}]}`
	config, err := DecodeConfig(bytes.NewReader([]byte(input)))

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(config.Deployments) != 1 {
		t.Fatalf("no deployments, expected 1")
	} else {
		depl := config.Deployments[0]

		if depl.Prefix != "myapp/production/versions" {
			t.Fatalf("bad prefix, got '%s', expected '%s'", depl.Prefix, "myapp/production/versions")
		}
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
		Server: "127.0.0.1:8000",
	}

	c2 := Config{
		WaitTime: 12 * time.Second,
	}

	c3 := Merge(&c1, &c2)

	if c3.Server != "127.0.0.1:8000" {
		t.Fatal("bad server field")
	}

	if c3.WaitTime != 12*time.Second {
		t.Fatal("bad waitTime field")
	}
}
