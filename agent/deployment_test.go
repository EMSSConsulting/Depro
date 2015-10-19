package agent

import (
	"testing"
)

func TestDeployment_FullPath(t *testing.T) {
	d := Deployment{
		Config: &DeploymentConfig{
			ID:   "test",
			Path: "/data/deploy",
		},
		agentConfig: &Config{
			Name: "test",
		},
	}

	if d.fullPath("") != "/data/deploy" {
		t.Fatalf("Expected fullPath(\"\") to be '/data/deploy', got '%s'", d.fullPath(""))
	}

	if d.fullPath("1234") != "/data/deploy/1234" {
		t.Fatalf("Expected fullPath(\"1234\") to be '/data/deploy/1234', got '%s'", d.fullPath("1234"))
	}
}

func TestDeployment_VersionPrefix(t *testing.T) {
	d := Deployment{
		Config: &DeploymentConfig{
			ID:     "test",
			Prefix: "deploy/myapp",
		},
		agentConfig: &Config{
			Name: "test",
		},
	}

	if d.versionPrefix("1234") != "deploy/myapp/1234" {
		t.Fatalf("Expected versionPrefix('%s') to be 'deploy/myapp/1234', got '%s'", d.versionPrefix("1234"))
	}
}
