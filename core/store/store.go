package store

import (
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"

	ma "github.com/multiformats/go-multiaddr"
)

// PeerStore is an interface that combines AddressBook and ProtocolBook.
type PeerStore interface {
	AddressBook
	ProtocolBook
}

// AddressBook is a store that manage the net addresses of peers.
type AddressBook interface {
	// AddAddress appends some net addresses of a peer.
	AddAddress(pid peer.ID, addr ...ma.Multiaddr) error

	// SetAddresses records some addresses of a peer.
	// This function will remove all addresses that are not in the list.
	SetAddresses(pid peer.ID, addresses []ma.Multiaddr) error

	// RemoveAddress removes some net addresses of a peer.
	RemoveAddress(pid peer.ID, addr ...ma.Multiaddr) error

	// GetFirstAddress returns the first net address of a peer. If no address is stored, it returns nil.
	GetFirstAddress(pid peer.ID) ma.Multiaddr

	// GetAddresses returns all net addresses of a peer.
	GetAddresses(pid peer.ID) []ma.Multiaddr
}

// ProtocolBook is a store that manages the protocols supported by peers.
type ProtocolBook interface {
	// AddProtocol appends some protocols supported by a peer.
	AddProtocol(pid peer.ID, protocols ...protocol.ID) error

	// SetProtocols records some protocols supported by a peer.
	// This function will remove all protocols that are not in the list.
	SetProtocols(pid peer.ID, protocols []protocol.ID) error

	// DeleteProtocol removes some protocols of a peer.
	DeleteProtocol(pid peer.ID, protocols ...protocol.ID) error

	// ClearProtocol removes all records of a peer.
	ClearProtocol(pid peer.ID) error

	// GetProtocols returns the list of protocols supported by a peer.
	GetProtocols(pid peer.ID) []protocol.ID

	// ContainsProtocol returns whether a peer has supported a specific protocol.
	ContainsProtocol(pid peer.ID, protocol protocol.ID) bool

	// ProtocolsContained returns the list of protocols supported by a peer that are contained in the given list.
	ProtocolsContained(pid peer.ID, protocols ...protocol.ID) []protocol.ID

	// PeersSupportingAllProtocols returns the list of peer IDs that support all the given protocols.
	PeersSupportingAllProtocols(protocols ...protocol.ID) []peer.ID
}
