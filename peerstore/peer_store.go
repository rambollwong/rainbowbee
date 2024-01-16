package peerstore

import (
	"sync"

	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/store"
	"github.com/rambollwong/rainbowcat/types"
)

var _ store.PeerStore = (*BasicPeerStore)(nil)

// BasicPeerStore is a basic implementation of the PeerStore interface.
type BasicPeerStore struct {
	store.ProtocolBook
	store.AddressBook
}

// NewBasicPeerStore creates a new instance of BasicPeerStore.
func NewBasicPeerStore(localPID peer.ID) store.PeerStore {
	return &BasicPeerStore{
		ProtocolBook: &protocolBook{
			mu:       sync.RWMutex{},
			book:     make(map[peer.ID]*types.Set[protocol.ID]),
			localPID: localPID,
		},
		AddressBook: &addressBook{
			book: sync.Map{},
		},
	}
}
