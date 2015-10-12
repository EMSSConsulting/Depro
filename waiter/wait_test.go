package waiter

import (
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func TestWaiter_SingleNodeNoState(t *testing.T) {

	apiConfig := api.DefaultConfig()

	apiConfig.Address = "127.0.0.1:8500"
	apiConfig.WaitTime = 10 * time.Second

	client, _ := api.NewClient(apiConfig)

	go func() {
		kv := client.KV()
		_, err := kv.DeleteTree("wait", nil)
		if err != nil {
			t.Fatalf("Failed to remove test tree node from Consul: %s", err.Error())
		}
	}()

	wait := NewWait(client, "wait", 1, nil)

	go func() {
		_, err := wait.Wait(10 * time.Second)
		if err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		kv := client.KV()

		p := &api.KVPair{
			Key:   "wait/gr1",
			Value: []byte("ready"),
		}

		time.Sleep(50 * time.Millisecond)
		kv.Put(p, nil)
	}()

	go func() {
		select {
		case update := <-wait.NodeUpdate():
			if update.Node != "gr1" {
				t.Fatalf("Expected first node update to be from gr1")
			}

			if update.State != "ready" {
				t.Fatalf("Expected initial gr1 state to be 'ready'")
			}

			if update.LastState != "" {
				t.Fatalf("LastState should be empty for a new entry")
			}
		case <-time.After(time.Second * 1):
			t.Fatalf("Expected node update channel to trigger")
		}
	}()

	go func() {
		select {
		case node := <-wait.NodeReady():
			if node.Node != "gr1" {
				t.Fatalf("Expected first node update to be from gr1")
			}

			if node.State != "ready" {
				t.Fatalf("Expected initial gr1 state to be 'ready'")
			}
		case <-time.After(time.Second * 1):
			t.Fatalf("Expected node ready channel to trigger")
		}
	}()

	select {
	case allReady := <-wait.AllReady():
		if len(allReady) != 1 {
			t.Fatalf("Expected one node to fulfil the allReady promise")
		}
	case <-time.After(time.Second * 1):
		t.Fatalf("Expected all nodes to be ready")
	}
}

func TestWaiter_SingleNodeState(t *testing.T) {

	apiConfig := api.DefaultConfig()

	apiConfig.Address = "127.0.0.1:8500"
	apiConfig.WaitTime = 1 * time.Second

	client, _ := api.NewClient(apiConfig)

	kv := client.KV()
	_, err := kv.DeleteTree("wait", nil)
	if err != nil {
		t.Fatalf("Failed to remove test tree node from Consul: %s", err.Error())
	}

	wait := NewWait(client, "wait", 1, func(w *WaitNode) bool {
		return w.State == "ready"
	})

	go func() {
		_, err := wait.Wait(10 * time.Second)
		if err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		kv := client.KV()

		p := &api.KVPair{
			Key:   "wait/gr1",
			Value: []byte("busy"),
		}

		time.Sleep(50 * time.Millisecond)
		kv.Put(p, nil)

		time.Sleep(50 * time.Millisecond)
		p.Value = []byte("ready")
		kv.Put(p, nil)
	}()

	go func() {
		select {
		case update := <-wait.NodeUpdate():
			if update.Node != "gr1" {
				t.Fatalf("Expected first node update to be from gr1")
			}

			if update.State != "busy" {
				t.Fatalf("Expected initial gr1 state to be 'busy'")
			}

			if update.LastState != "" {
				t.Fatalf("LastState should be empty for a new entry")
			}
		case <-time.After(time.Second * 10):
			t.Fatalf("Expected node update channel to trigger")
		}

		select {
		case update := <-wait.NodeUpdate():
			if update.Node != "gr1" {
				t.Fatalf("Expected second node update to be from gr1")
			}

			if update.State != "ready" {
				t.Fatalf("Expected updated gr1 state to be 'ready'")
			}

			if update.LastState != "busy" {
				t.Fatalf("LastState should be 'busy' for a gr1")
			}
		case <-time.After(time.Second * 10):
			t.Fatalf("Expected node update channel to trigger")
		}
	}()

	go func() {
		select {
		case node := <-wait.NodeReady():
			if node.Node != "gr1" {
				t.Fatalf("Expected first node ready to be from gr1")
			}

			if node.State != "ready" {
				t.Fatalf("Expected gr1 state to be 'ready'")
			}
		case <-time.After(time.Second * 10):
			t.Fatalf("Expected node ready channel to trigger")
		}
	}()

	select {
	case allReady := <-wait.AllReady():
		if len(allReady) != 1 {
			t.Fatalf("Expected one node to fulfil the allReady promise")
		}
	case <-time.After(time.Second * 10):
		t.Fatalf("Expected all nodes to be ready")
	}
}
