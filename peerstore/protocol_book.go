package peerstore

import (
	"sync"

	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/store"
	"github.com/rambollwong/rainbowcat/types"
)

var _ store.ProtocolBook = (*protocolBook)(nil)

type protocolBook struct {
	mu       sync.RWMutex
	book     map[peer.ID]*types.Set[protocol.ID]
	localPID peer.ID
}

// initialOrGetProtocolSet returns the protocol set for a peer ID. If the set does not exist, it creates a new one.
func (p *protocolBook) initialOrGetProtocolSet(peerID peer.ID) *types.Set[protocol.ID] {
	set, ok := p.book[peerID]
	if !ok {
		set = types.NewSet[protocol.ID]()
		p.book[peerID] = set
	}
	return set
}

// AddProtocol adds one or more protocols to the protocol set of a peer ID.
func (p *protocolBook) AddProtocol(peerID peer.ID, protocols ...protocol.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	set := p.initialOrGetProtocolSet(peerID)
	for _, protocolID := range protocols {
		set.Put(protocolID)
	}
	return nil
}

// SetProtocols sets the protocols for a peer ID, replacing any existing protocols.
func (p *protocolBook) SetProtocols(peerID peer.ID, protocols []protocol.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	set := types.NewSet[protocol.ID]()
	for _, protocolID := range protocols {
		set.Put(protocolID)
	}
	p.book[peerID] = set
	return nil
}

// DeleteProtocol removes one or more protocols from the protocol set of a peer ID.
func (p *protocolBook) DeleteProtocol(peerID peer.ID, protocols ...protocol.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	set, ok := p.book[peerID]
	if !ok {
		return nil
	}
	for _, protocolID := range protocols {
		set.Remove(protocolID)
	}
	return nil
}

// ClearProtocol removes all protocols associated with a peer ID.
func (p *protocolBook) ClearProtocol(peerID peer.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.book, peerID)
	return nil
}

// GetProtocols returns the protocols associated with a peer ID.
func (p *protocolBook) GetProtocols(peerID peer.ID) []protocol.ID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set, ok := p.book[peerID]
	if !ok {
		return nil
	}
	res := make([]protocol.ID, 0, set.Size())
	set.Range(func(protocolID protocol.ID) bool {
		res = append(res, protocolID)
		return true
	})
	return res
}

// ContainsProtocol checks if a peer ID has a specific protocol.
func (p *protocolBook) ContainsProtocol(peerID peer.ID, protocol protocol.ID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set, ok := p.book[peerID]
	if !ok {
		return false
	}
	return set.Exist(protocol)
}

// ProtocolsContained returns the protocols from a list that are associated with a peer ID.
func (p *protocolBook) ProtocolsContained(peerID peer.ID, protocols ...protocol.ID) (contained []protocol.ID) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set, ok := p.book[peerID]
	if !ok {
		return contained
	}
	for _, protocolID := range protocols {
		if set.Exist(protocolID) {
			contained = append(contained, protocolID)
		}
	}
	return contained
}

// PeersSupportingAllProtocols returns the peer IDs that support all the specified protocols.
func (p *protocolBook) PeersSupportingAllProtocols(protocols ...protocol.ID) (peers []peer.ID) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peers = make([]peer.ID, 0, len(p.book))
	for pid, set := range p.book {
		if pid == p.localPID {
			continue
		}
		containsAll := true
		for _, protocolID := range protocols {
			if !set.Exist(protocolID) {
				containsAll = false
				break
			}
		}
		if containsAll {
			peers = append(peers, pid)
		}
	}
	return peers
}
