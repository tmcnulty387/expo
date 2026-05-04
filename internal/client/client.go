// Package client models the internal state of an Expo client.
// Applications seeking to provide an Expo interface should create a [Client].
//
// Messages are communicated by:
// 1. Sending a valid message.Header.
// 2. Sending a stream of bytes, message.Header.Size bytes long.
//
// This stream of bytes can be deserialized into the type specified by
// MessageHeader.Kind.
package client

import (
	"context"
	"encoding/base32"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

type Client struct {
	host  host.Host
	peers []peer.ID
}

func NewClient() *Client {
	return &Client{}
}

// CreateSession starts a new session by creating a libp2p host and returns a
// join code that can be shared with other clients to join the session.
func (c *Client) CreateSession() (string, error) {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return "", err
	}

	log.Printf("Node created with ID %s and addresses: %v\n", node.ID(), node.Addrs())

	c.host = node
	c.peers = []peer.ID{}

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
func (c *Client) JoinSession(joinCode string) error {
	addrBytes, err := base32.StdEncoding.DecodeString(joinCode)
	if err != nil {
		return fmt.Errorf("failed to decode join code: %w", err)
	}

	ma, err := multiaddr.NewMultiaddrBytes(addrBytes)
	if err != nil {
		return fmt.Errorf("failed to parse join code: %w", err)
	}

	hostInfo, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return fmt.Errorf("failed to extract peer info from join code: %w", err)
	}

	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	node.Peerstore().AddAddrs(hostInfo.ID, hostInfo.Addrs, peerstore.PermanentAddrTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := node.Connect(ctx, *hostInfo); err != nil {
		node.Close()
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	if node.Network().Connectedness(hostInfo.ID) != network.Connected {
		node.Close()
		return fmt.Errorf("not connected to peer after Connect call")
	}

	c.host = node
	c.peers = []peer.ID{hostInfo.ID}

	// TODO: Announce ourselves so the creator introduces us to all existing peers.

	log.Printf("Joined session, connected to %s\n", hostInfo.ID)
	return nil
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
