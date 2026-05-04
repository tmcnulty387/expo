package client

import (
	"encoding/base32"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestCreateSession(t *testing.T) {
	c := NewClient()
	joinCode, err := c.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	if joinCode == "" {
		t.Fatal("expected non-empty join code")
	}
	t.Logf("Generated join code: %s", joinCode)
}

func TestJoinSession(t *testing.T) {
	c := NewClient()
	// Create a host that will act as the session host
	hostNode, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer hostNode.Close()

	// Get the host's addresses
	peerInfo := peer.AddrInfo{
		ID:    hostNode.ID(),
		Addrs: hostNode.Addrs(),
	}
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		t.Fatal(err)
	}

	// Use all addresses (should include localhost for testing)
	if len(addrs) == 0 {
		t.Fatal("no addresses found")
	}
	t.Logf("Host addresses: %v", addrs)

	// Create a join code from all addresses
	joinCode := encodeJoinCode(addrs)
	t.Logf("Join code: %s", joinCode)

	// Now try to join using the join code
	clientNode, err := c.JoinSession(joinCode)
	if err != nil {
		t.Fatal(err)
	}
	defer clientNode.Close()

	// Verify connection
	if clientNode.Network().Connectedness(hostNode.ID()) != network.Connected {
		t.Fatal("client not connected to host")
	}

	t.Logf("Successfully connected client %s to host %s", clientNode.ID(), hostNode.ID())
}

// Helper function to encode multiaddrs as a join code
func encodeJoinCode(addrs []multiaddr.Multiaddr) string {
	var addrStrings []string
	for _, addr := range addrs {
		addrStrings = append(addrStrings, addr.String())
	}
	allAddrs := strings.Join(addrStrings, "|")
	return base32.StdEncoding.EncodeToString([]byte(allAddrs))
}