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
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

// TODO: Command struct that can be used for CLI/GUI to communicate with its
// Client?

type Client struct {
	// TODO: fields?
}

func CreateSession() (string, error) {
	// start a libp2p node with default settings
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return "", err
	}

	// Build AddrInfo
	peerInfo := peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		node.Close()
		return "", err
	}
	log.Printf("Peer Info: %v\n", peerInfo)
	log.Printf("Peer Addresses: %v\n", addrs)

	// Encode all addresses (let libp2p handle choosing the best one)
	// Join all multiaddr strings with "|" separator
	var addrStrings []string
	for _, addr := range addrs {
		addrStrings = append(addrStrings, addr.String())
	}
	allAddrs := strings.Join(addrStrings, "|")

	// Encode as base32 for the join code
	joinCode := base32.StdEncoding.EncodeToString([]byte(allAddrs))

	log.Printf("Join Code: %s\n", joinCode)
	log.Printf("Encoded %d addresses\n", len(addrs))

	// TODO: Keep the node running
	if err := node.Close(); err != nil {
		return "", err
	}

	return joinCode, nil
}

// JoinSession decodes a base32 join code and connects to the specified peer.
// Returns the connected libp2p host for further communication.
func JoinSession(joinCode string) (host.Host, error) {
	// Decode the base32 join code
	addrBytes, err := base32.StdEncoding.DecodeString(joinCode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode join code: %w", err)
	}

	// Split the addresses by "|" separator
	allAddrs := string(addrBytes)
	addrStrings := strings.Split(allAddrs, "|")

	if len(addrStrings) == 0 {
		return nil, fmt.Errorf("no addresses found in join code")
	}

	log.Printf("Decoded %d addresses\n", len(addrStrings))

	// Parse all multiaddrs and extract peer info
	var peerInfo *peer.AddrInfo
	var addrs []multiaddr.Multiaddr

	for _, addrStr := range addrStrings {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("Warning: failed to parse address %s: %v\n", addrStr, err)
			continue
		}

		// Extract peer info from the first valid address
		if peerInfo == nil {
			info, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				log.Printf("Warning: failed to extract peer info from %s: %v\n", addrStr, err)
				continue
			}
			peerInfo = info
			addrs = append(addrs, info.Addrs...)
		} else {
			// For subsequent addresses, just extract the address part
			info, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				log.Printf("Warning: failed to extract peer info from %s: %v\n", addrStr, err)
				continue
			}
			addrs = append(addrs, info.Addrs...)
		}
	}

	if peerInfo == nil {
		return nil, fmt.Errorf("failed to extract peer info from any address")
	}

	// Use all the addresses we collected
	peerInfo.Addrs = addrs

	log.Printf("Connecting to peer: %s at %v\n", peerInfo.ID, peerInfo.Addrs)

	// Create a new libp2p node
	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Add the peer to the peerstore
	node.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

	// Connect to the peer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := node.Connect(ctx, *peerInfo); err != nil {
		node.Close()
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	// Verify connection
	if node.Network().Connectedness(peerInfo.ID) != network.Connected {
		node.Close()
		return nil, fmt.Errorf("not connected to peer after Connect call")
	}

	log.Printf("Successfully connected to peer %s\n", peerInfo.ID)

	return node, nil
}
