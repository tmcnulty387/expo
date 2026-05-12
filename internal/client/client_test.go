package client

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
	"github.com/tmcnulty387/expo/internal/client/message"
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

// TestMessagesChannelDelivery verifies that draw messages sent by one peer
// arrive in the other peer's Messages channel, and that peer-management
// messages (PeerAnnounce, PeerList, PeerIntroduction) do not.
func TestMessagesChannelDelivery(t *testing.T) {
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

	// Wait for host to register the joiner.
	if !waitFor(t, 2*time.Second, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return len(host.peers) == 1
	}) {
		t.Fatal("host did not register joiner in time")
	}

	// Drain any messages already queued from the handshake.
	for len(host.Messages) > 0 {
		<-host.Messages
	}
	for len(joiner.Messages) > 0 {
		<-joiner.Messages
	}

	sent := &message.Echo{Text: "hello from joiner"}
	if err := joiner.SendMessage(host.host.ID(), sent); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	select {
	case got := <-host.Messages:
		echo, ok := got.(*message.Echo)
		if !ok {
			t.Fatalf("expected *message.Echo, got %T", got)
		}
		if echo.Text != sent.Text {
			t.Fatalf("expected %q, got %q", sent.Text, echo.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message did not arrive in Messages channel within timeout")
	}

	// Peer-management messages must not be in the channel.
	select {
	case leaked := <-host.Messages:
		t.Fatalf("peer-management message leaked into Messages channel: %T", leaked)
	default:
	}
}

func TestPickWANAddr(t *testing.T) {
	mustMA := func(s string) multiaddr.Multiaddr {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			t.Fatalf("bad multiaddr %q: %v", s, err)
		}
		return ma
	}

	loopback := mustMA("/ip4/127.0.0.1/tcp/4001")
	wan := mustMA("/ip4/1.2.3.4/tcp/4001")
	loopback6 := mustMA("/ip6/::1/tcp/4001")

	if pickWANAddr(nil) != nil {
		t.Error("expected nil for empty input")
	}
	if pickWANAddr([]multiaddr.Multiaddr{}) != nil {
		t.Error("expected nil for empty slice")
	}

	// All loopback: falls back to first.
	got := pickWANAddr([]multiaddr.Multiaddr{loopback, loopback6})
	if got == nil || got.String() != loopback.String() {
		t.Errorf("all-loopback fallback: got %v, want %v", got, loopback)
	}

	// WAN address is preferred over loopback.
	got = pickWANAddr([]multiaddr.Multiaddr{loopback, wan})
	if got == nil || got.String() != wan.String() {
		t.Errorf("wan preference: got %v, want %v", got, wan)
	}

	// Verify the picked address is not loopback.
	ip, _ := got.ValueForProtocol(multiaddr.P_IP4)
	if parsed := net.ParseIP(ip); parsed == nil || parsed.IsLoopback() {
		t.Errorf("picked address %v is loopback", got)
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
