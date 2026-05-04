package client

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
)

func TestCreateSession(t *testing.T) {
	client := NewClient()
	defer client.Close()

	joinCode, err := client.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	if joinCode == "" {
		t.Fatal("expected non-empty join code")
	}
	t.Logf("Generated join code: %s", joinCode)
}

func TestJoinSession(t *testing.T) {
	hostClient := NewClient()
	defer hostClient.Close()

	joinCode, err := hostClient.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Host join code: %s", joinCode)

	joiningClient := NewClient()
	defer joiningClient.Close()

	err = joiningClient.JoinSession(joinCode)
	if err != nil {
		t.Fatal(err)
	}

	if len(joiningClient.peers) == 0 {
		t.Fatal("no peers connected")
	}

	hostID := joiningClient.peers[0]
	if joiningClient.host.Network().Connectedness(hostID) != network.Connected {
		t.Fatal("client not connected to host")
	}

	t.Logf("Successfully connected client %s to host %s", joiningClient.host.ID(), hostID)
}
