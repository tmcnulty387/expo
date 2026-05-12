package client

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/tmcnulty387/expo/internal/canvas"
	"github.com/tmcnulty387/expo/internal/client/message"
)

// hasPeer reports whether id is already in c.peers. Caller must hold c.mu.
func (c *Client) hasPeer(id peer.ID) bool {
	for _, p := range c.peers {
		if p == id {
			return true
		}
	}
	return false
}

// peersExcluding returns a snapshot of c.peers with id removed. Caller must hold c.mu.
func (c *Client) peersExcluding(id peer.ID) []peer.ID {
	result := make([]peer.ID, 0, len(c.peers))
	for _, p := range c.peers {
		if p != id {
			result = append(result, p)
		}
	}
	return result
}

// RegisterStreamHandler sets up the protocol handler to receive incoming messages
// and a network notifier that tracks peers as they connect.
func (c *Client) RegisterStreamHandler() {
	if c.host == nil {
		return
	}
	c.host.SetStreamHandler(ProtocolID, c.handleStream)
	c.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			remotePeer := conn.RemotePeer()
			c.mu.Lock()
			if !c.hasPeer(remotePeer) {
				c.peers = append(c.peers, remotePeer)
			}
			c.mu.Unlock()
		},
	})
}

// handleStream reads one message from the stream and dispatches it.
func (c *Client) handleStream(stream network.Stream) {
	defer stream.Close()
	msg, err := message.Read(stream)
	if err != nil {
		log.Printf("error reading message from %s: %v", stream.Conn().RemotePeer(), err)
		return
	}
	// Peer-management messages are handled internally and must not reach the UI.
	switch m := msg.(type) {
	case *message.PeerAnnounce:
		c.handlePeerAnnounce(stream.Conn().RemotePeer(), m)
		return
	case *message.PeerList:
		c.handlePeerList(m)
		return
	case *message.PeerIntroduction:
		c.handlePeerIntroduction(m)
		return
	}
	select {
	case c.Messages <- msg:
	default:
		log.Printf("Messages channel full, dropping %T from %s", msg, stream.Conn().RemotePeer())
	}
}

// handlePeerAnnounce is called on the session creator when a joiner connects.
// It sends the current peer list to the joiner and introduces the joiner to
// all existing peers.
func (c *Client) handlePeerAnnounce(joinerID peer.ID, ann *message.PeerAnnounce) {
	c.mu.Lock()
	// ConnectedF may have already added joinerID; exclude it from the "existing
	// peers" snapshot so we don't send the joiner their own address.
	existing := c.peersExcluding(joinerID)
	peerAddrs := make([]string, 0, len(existing))
	for _, p := range existing {
		info := peer.AddrInfo{ID: p, Addrs: c.host.Peerstore().Addrs(p)}
		p2pAddrs, err := peer.AddrInfoToP2pAddrs(&info)
		if err != nil {
			continue
		}
		if chosen := pickWANAddr(p2pAddrs); chosen != nil {
			peerAddrs = append(peerAddrs, chosen.String())
		}
	}
	if !c.hasPeer(joinerID) {
		c.peers = append(c.peers, joinerID)
	}
	c.mu.Unlock()

	if err := c.SendMessage(joinerID, &message.PeerList{Addrs: peerAddrs}); err != nil {
		log.Printf("failed to send peer list to %s: %v", joinerID, err)
	}

	msgs := canvas.Snapshot()
	for _, msg := range msgs {
		if err := c.SendMessage(joinerID, msg); err != nil {
			log.Printf("failed to send canvas snapshot message to %s: %v", joinerID, err)
		}
	}

	joinerInfo := peer.AddrInfo{ID: joinerID, Addrs: c.host.Peerstore().Addrs(joinerID)}
	joinerP2PAddrs, err := peer.AddrInfoToP2pAddrs(&joinerInfo)
	if err != nil {
		log.Printf("failed to build joiner p2p addrs: %v", err)
		return
	}
	introAddrs := make([]string, 0, len(joinerP2PAddrs))
	for _, a := range joinerP2PAddrs {
		introAddrs = append(introAddrs, a.String())
	}
	intro := &message.PeerIntroduction{Addrs: introAddrs}
	for _, p := range existing {
		if err := c.SendMessage(p, intro); err != nil {
			log.Printf("failed to send peer introduction to %s: %v", p, err)
		}
	}
}

// handlePeerList is called on the joiner after receiving the existing peer list
// from the session creator. The joiner dials each listed peer; ConnectedF on
// both sides then registers the new peer in c.peers.
func (c *Client) handlePeerList(list *message.PeerList) {
	for _, addrStr := range list.Addrs {
		ma, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("invalid addr in peer list: %v", err)
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Printf("invalid peer info in peer list: %v", err)
			continue
		}
		c.connectToPeer(info)
	}
}

// handlePeerIntroduction is called on existing peers when the session creator
// introduces a newly joined peer. We add the peer's addresses to the peerstore
// so we can accept their incoming connection; ConnectedF will register them in
// c.peers once they dial us. We intentionally do not dial here to avoid a
// simultaneous-open TLS conflict with the joiner's outbound dial via PeerList.
func (c *Client) handlePeerIntroduction(intro *message.PeerIntroduction) {
	for _, addrStr := range intro.Addrs {
		ma, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("invalid addr in peer introduction: %v", err)
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Printf("invalid peer info in peer introduction: %v", err)
			continue
		}
		c.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	}
}

// connectToPeer attempts a direct connection, then falls back to a circuit
// relay through the session creator if direct fails.
func (c *Client) connectToPeer(info *peer.AddrInfo) {
	c.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.host.Connect(ctx, *info); err == nil {
		log.Printf("Connected to peer %s (direct)", info.ID)
		return
	} else {
		log.Printf("Direct connect to %s failed: %v; trying relay", info.ID, err)
	}

	if c.relayPeer == "" {
		log.Printf("No relay peer configured, giving up on %s", info.ID)
		return
	}

	relayAddrs := c.host.Peerstore().Addrs(c.relayPeer)
	circuitAddrs := buildCircuitAddrs(c.relayPeer, relayAddrs, info.ID)
	if len(circuitAddrs) == 0 {
		log.Printf("Could not build relay addrs for %s", info.ID)
		return
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	relayedInfo := peer.AddrInfo{ID: info.ID, Addrs: circuitAddrs}
	if err := c.host.Connect(ctx2, relayedInfo); err != nil {
		log.Printf("Relay connect to %s failed: %v", info.ID, err)
		return
	}
	log.Printf("Connected to peer %s (via relay %s)", info.ID, c.relayPeer)
}

// buildCircuitAddrs constructs circuit relay multiaddrs for targetID routed
// through the relay node at relayID/relayAddrs. One addr is produced per
// relay transport address.
func buildCircuitAddrs(relayID peer.ID, relayAddrs []multiaddr.Multiaddr, targetID peer.ID) []multiaddr.Multiaddr {
	relayP2P, err := multiaddr.NewMultiaddr("/p2p/" + relayID.String())
	if err != nil {
		return nil
	}
	circuit, err := multiaddr.NewMultiaddr("/p2p-circuit")
	if err != nil {
		return nil
	}
	targetP2P, err := multiaddr.NewMultiaddr("/p2p/" + targetID.String())
	if err != nil {
		return nil
	}

	addrs := make([]multiaddr.Multiaddr, 0, len(relayAddrs))
	for _, addr := range relayAddrs {
		addrs = append(addrs, addr.Encapsulate(relayP2P).Encapsulate(circuit).Encapsulate(targetP2P))
	}
	return addrs
}
