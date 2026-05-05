package client

import (
	"context"
	"testing"
	"time"

	"github.com/Go-20255/team-project-malloc4/internal/client/message"
	"github.com/libp2p/go-libp2p/core/network"
)

// waitFor polls cond every 50ms until it returns true or timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

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

	localCode, err := joiningClient.JoinSession(joinCode)
	if err != nil {
		t.Fatal(err)
	}
	if localCode == "" {
		t.Fatal("expected non-empty local join code")
	}
	t.Logf("Joining client local join code: %s", localCode)

	if len(joiningClient.peers) == 0 {
		t.Fatal("joiner has no peers")
	}

	hostID := joiningClient.peers[0]
	if joiningClient.host.Network().Connectedness(hostID) != network.Connected {
		t.Fatal("joiner not connected to host")
	}

	// PeerAnnounce is handled asynchronously; wait for the host to register the joiner.
	if !waitFor(t, 2*time.Second, func() bool {
		hostClient.mu.Lock()
		defer hostClient.mu.Unlock()
		return len(hostClient.peers) == 1
	}) {
		t.Fatal("host did not register joiner in time")
	}

	joinerID := joiningClient.host.ID()
	if hostClient.host.Network().Connectedness(joinerID) != network.Connected {
		t.Fatal("host not connected to joiner")
	}

	t.Logf("Successfully connected client %s to host %s", joinerID, hostID)
}

func TestGenerateJoinCode(t *testing.T) {
	c := NewClient()
	defer c.Close()

	if _, err := c.GenerateJoinCode(); err == nil {
		t.Fatal("expected error when not connected")
	}

	if _, err := c.CreateSession(); err != nil {
		t.Fatal(err)
	}

	code, err := c.GenerateJoinCode()
	if err != nil {
		t.Fatal(err)
	}
	if code == "" {
		t.Fatal("expected non-empty join code")
	}
}

func TestRegisterStreamHandler(t *testing.T) {
	host := NewClient()
	defer host.Close()

	joinCode, err := host.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	joiner := NewClient()
	defer joiner.Close()

	if _, err := joiner.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := joiner.host.NewStream(ctx, host.host.ID(), ProtocolID)
	if err != nil {
		t.Fatalf("failed to open stream to registered handler: %v", err)
	}
	stream.Close()
}

func TestSendMessage(t *testing.T) {
	host := NewClient()
	defer host.Close()

	joinCode, err := host.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	joiner := NewClient()
	defer joiner.Close()

	if _, err := joiner.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}

	msg := &message.Echo{Text: "hello"}
	if err := joiner.SendMessage(host.host.ID(), msg); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
}

func TestBroadcastMessage(t *testing.T) {
	creator := NewClient()
	defer creator.Close()

	joinCode, err := creator.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	joiner1 := NewClient()
	defer joiner1.Close()
	joiner2 := NewClient()
	defer joiner2.Close()

	if _, err := joiner1.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}
	if _, err := joiner2.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}

	// Wait for creator to register both joiners before broadcasting.
	if !waitFor(t, 2*time.Second, func() bool {
		creator.mu.Lock()
		defer creator.mu.Unlock()
		return len(creator.peers) == 2
	}) {
		t.Fatal("creator did not register both joiners in time")
	}

	msg := &message.Echo{Text: "broadcast test"}
	if err := creator.BroadcastMessage(msg); err != nil {
		t.Fatalf("BroadcastMessage failed: %v", err)
	}
}

func TestPeerDiscovery(t *testing.T) {
	a := NewClient()
	defer a.Close()

	joinCode, err := a.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	b := NewClient()
	defer b.Close()
	if _, err := b.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}

	c := NewClient()
	defer c.Close()
	if _, err := c.JoinSession(joinCode); err != nil {
		t.Fatal(err)
	}

	isConnected := func(from, to *Client) bool {
		return from.host.Network().Connectedness(to.host.ID()) == network.Connected
	}

	// Wait for full mesh: every pair must be directly connected.
	if !waitFor(t, 3*time.Second, func() bool {
		return isConnected(a, b) && isConnected(a, c) &&
			isConnected(b, a) && isConnected(b, c) &&
			isConnected(c, a) && isConnected(c, b)
	}) {
		t.Errorf("a <-> b connected: %v", isConnected(a, b))
		t.Errorf("a <-> c connected: %v", isConnected(a, c))
		t.Errorf("b <-> c connected: %v", isConnected(b, c))
		t.Fatal("full mesh not established within timeout")
	}

	// Verify peer list lengths.
	check := func(name string, cl *Client, want int) {
		t.Helper()
		cl.mu.Lock()
		got := len(cl.peers)
		cl.mu.Unlock()
		if got != want {
			t.Errorf("%s has %d peers, want %d", name, got, want)
		}
	}
	check("a", a, 2)
	check("b", b, 2)
	check("c", c, 2)
}
