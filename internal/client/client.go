// Package client models the internal state of an Expo client.
// Applications seeking to provide an Expo interface should create a [Client].
package client

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Go-20255/team-project-malloc4/internal/client/message"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

const ProtocolID = protocol.ID("/expo/1.0.0")

type Client struct {
	host      host.Host
	peers     []peer.ID
	relayPeer peer.ID // session creator; used as circuit relay fallback
	mu        sync.Mutex
	Messages  chan message.Message // stream handler -> UI
}

func NewClient() *Client {
	return &Client{Messages: make(chan message.Message, 32)}
}

// CreateSession starts a new session by creating a libp2p host and returns a
// join code that can be shared with other clients to join the session.
func (c *Client) CreateSession() (string, error) {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelayService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return "", err
	}

	log.Printf("Node created with ID %s and addresses: %v\n", node.ID(), node.Addrs())

	c.host = node
	c.peers = []peer.ID{}
	c.RegisterStreamHandler()

	joinCode, err := c.GenerateJoinCode()
	if err != nil {
		node.Close()
		return "", err
	}

	log.Printf("Session created, join code: %s\n", joinCode)
	return joinCode, nil
}

// JoinSession decodes a join code, connects to the session creator, and sends
// a PeerAnnounce so the creator can introduce all existing peers.
// Returns this client's own join code so others can connect to it.
func (c *Client) JoinSession(joinCode string) (string, error) {
	addrBytes, err := base32.StdEncoding.DecodeString(joinCode)
	if err != nil {
		return "", fmt.Errorf("failed to decode join code: %w", err)
	}

	ma, err := multiaddr.NewMultiaddrBytes(addrBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse join code: %w", err)
	}

	hostInfo, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return "", fmt.Errorf("failed to extract peer info from join code: %w", err)
	}

	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	node.Peerstore().AddAddrs(hostInfo.ID, hostInfo.Addrs, peerstore.PermanentAddrTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := node.Connect(ctx, *hostInfo); err != nil {
		node.Close()
		return "", fmt.Errorf("failed to connect to peer: %w", err)
	}

	if node.Network().Connectedness(hostInfo.ID) != network.Connected {
		node.Close()
		return "", fmt.Errorf("not connected to peer after Connect call")
	}

	c.host = node
	c.peers = []peer.ID{hostInfo.ID}
	c.relayPeer = hostInfo.ID
	c.RegisterStreamHandler()

	wanAddr := pickWANAddr(node.Addrs())
	if wanAddr == nil {
		c.Close()
		return "", fmt.Errorf("no usable address found for peer announcement")
	}
	announce := &message.PeerAnnounce{Addr: wanAddr.String()}

	if err := c.SendMessage(hostInfo.ID, announce); err != nil {
		c.Close()
		return "", fmt.Errorf("failed to send peer announce: %w", err)
	}

	localCode, err := c.GenerateJoinCode()
	if err != nil {
		c.Close()
		return "", fmt.Errorf("failed to generate local join code: %w", err)
	}

	log.Printf("Joined session, connected to %s, local join code: %s\n", hostInfo.ID, localCode)
	return localCode, nil
}

// Close shuts down the client's host.
func (c *Client) Close() error {
	if c.host != nil {
		return c.host.Close()
	}
	return nil
}

// GenerateJoinCode produces a join code from this client's current address.
func (c *Client) GenerateJoinCode() (string, error) {
	if c.host == nil {
		return "", fmt.Errorf("client not connected")
	}
	peerInfo := peer.AddrInfo{ID: c.host.ID(), Addrs: c.host.Addrs()}
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		return "", err
	}
	chosen := pickWANAddr(addrs)
	if chosen == nil {
		return "", fmt.Errorf("no usable address found")
	}
	return base32.StdEncoding.EncodeToString(chosen.Bytes()), nil
}

const (
	maxSendRetries = 3
	retryDelay     = 100 * time.Millisecond
)

func (c *Client) SendMessage(peerID peer.ID, msg message.Message) error {
	if c.host == nil {
		return fmt.Errorf("client not connected")
	}

	var lastErr error
	for attempt := range maxSendRetries {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lastErr = func() error {
			defer cancel()
			stream, err := c.host.NewStream(ctx, peerID, ProtocolID)
			if err != nil {
				return fmt.Errorf("failed to open stream: %w", err)
			}
			defer stream.Close()
			if err := message.Write(stream, msg); err != nil {
				return fmt.Errorf("failed to write message: %w", err)
			}
			return nil
		}()

		if lastErr == nil {
			return nil
		}
		log.Printf("send to %s attempt %d/%d failed: %v", peerID, attempt+1, maxSendRetries, lastErr)
	}
	return lastErr
}

// BroadcastMessage sends a message to all connected peers concurrently.
func (c *Client) BroadcastMessage(msg message.Message) error {
	c.mu.Lock()
	peers := make([]peer.ID, len(c.peers))
	copy(peers, c.peers)
	c.mu.Unlock()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, p := range peers {
		wg.Add(1)
		go func(id peer.ID) {
			defer wg.Done()
			if err := c.SendMessage(id, msg); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("peer %s: %w", id, err))
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return errors.Join(errs...)
}

// pickWANAddr returns the first non-loopback address from addrs, falling back
// to the first address if all are loopback.
func pickWANAddr(addrs []multiaddr.Multiaddr) multiaddr.Multiaddr {
	for _, addr := range addrs {
		ip, err := addr.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			ip, err = addr.ValueForProtocol(multiaddr.P_IP6)
			if err != nil {
				continue
			}
		}
		if parsed := net.ParseIP(ip); parsed != nil && !parsed.IsLoopback() {
			return addr
		}
	}
	if len(addrs) > 0 {
		return addrs[0]
	}
	return nil
}
