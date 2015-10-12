package deploy

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

func TestProcess(t *testing.T) {
	config := &Config{
		Server:   "127.0.0.1:8500",
		Prefix:   "versions",
		WaitTime: 10 * time.Second,
		Nodes:    1,
	}

	apiConfig := api.DefaultConfig()

	apiConfig.Address = config.Server
	apiConfig.WaitTime = config.WaitTime

	client, _ := api.NewClient(apiConfig)

	ui := &cli.BasicUi{
		Writer: os.Stdout,
	}

	op := Operation{
		Version: "test",
		Config:  config,
		Client:  client,
		UI:      ui,
	}

	finishedCh := make(chan bool)

	kv := client.KV()
	_, err := kv.DeleteTree("versions", nil)
	if err != nil {
		t.Fatalf("Failed to remove test tree node from Consul: %s", err.Error())
	}

	go func() {
		err := op.Run()
		if err != nil {
			t.Fatal(err)
		}

		finishedCh <- true
	}()

	go func() {
		kv := client.KV()
		p := &api.KVPair{
			Key:   "versions/test/node1",
			Value: []byte("busy"),
		}

		time.Sleep(time.Millisecond * 100)
		_, err := kv.Put(p, nil)
		if err != nil {
			t.Fatal(fmt.Errorf("Failed to set key: %s", err.Error()))
		}

		time.Sleep(time.Millisecond * 50)
		p.Value = []byte("ready")
		_, err = kv.Put(p, nil)
		if err != nil {
			t.Fatal(fmt.Errorf("Failed to set key: %s", err.Error()))
		}
	}()

	<-finishedCh
}
