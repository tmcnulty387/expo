package client

import (
	"context"
	"log"
	"time"

	"github.com/Go-20255/team-project-malloc4/internal/client/message"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
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
	// Special internal handling of some messages.
	switch m := msg.(type) {
	case *message.PeerAnnounce:
		c.handlePeerAnnounce(stream.Conn().RemotePeer(), m)
	case *message.PeerList:
		c.handlePeerList(m)
	case *message.PeerIntroduction:
		c.handlePeerIntroduction(m)
	}
	c.Messages <- msg
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

// connectToPeer connects to a peer and adds it to the peer list.
func (c *Client) connectToPeer(info *peer.AddrInfo) {
	c.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.host.Connect(ctx, *info); err != nil {
		log.Printf("failed to connect to peer %s: %v", info.ID, err)
		return
	}
	log.Printf("Connected to peer %s", info.ID)
}
