package mgr

import (
	"io"

	"rambollwong/rainbowbee/core"
	"rambollwong/rainbowbee/core/network"
	"rambollwong/rainbowbee/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// ConnectionMgr provides a connection manager.
type ConnectionMgr interface {
	io.Closer // ConnectionMgr interface inherits the io.Closer interface, indicating it can be closed.

	// AddPeerConnection adds the given peer.ID and network.Connection to the connection manager.
	// It returns a boolean indicating whether the connection was successfully added.
	AddPeerConnection(pid peer.ID, conn network.Connection) bool

	// RemovePeerConnection removes the given peer.ID and network.Connection from the connection manager.
	// It returns a boolean indicating whether the connection was successfully removed.
	RemovePeerConnection(pid peer.ID, conn network.Connection) bool

	// ExistPeerConnection checks if the given peer.ID and network.Connection exist in the connection manager.
	// It returns a boolean indicating whether the given connection exists.
	ExistPeerConnection(pid peer.ID, conn network.Connection) bool

	// GetPeerConnection returns the network.Connection for the given peer.ID.
	GetPeerConnection(pid peer.ID) network.Connection

	// GetPeerAllConnection returns all network.Connections for the given peer.ID.
	GetPeerAllConnection(pid peer.ID) []network.Connection

	// IsConnected checks if the given peer.ID is connected.
	// It returns a boolean indicating whether the peer is connected.
	IsConnected(pid peer.ID) bool

	// IsAllowed checks if the given peer.ID is allowed.
	// It returns a boolean indicating whether the peer is allowed.
	IsAllowed(pid peer.ID) bool

	// ExpendConnection expands the number of connections for the given peer.ID.
	ExpendConnection(pid peer.ID)

	// MaxPeerCountAllowed returns the maximum number of allowed peers.
	MaxPeerCountAllowed() int

	// PeerCount returns the current number of connected peers.
	PeerCount() int

	// AllPeer returns a list of all peer.IDs.
	AllPeer() []peer.ID
}

// ConnSupervisor maintains the connection state of the necessary peers.
// If a necessary peer is not connected to us, supervisor will try to dial to it.
type ConnSupervisor interface {
	core.Switcher

	// SetPeerAddr will set a peer as a necessary peer and store the peer's address.
	SetPeerAddr(pid peer.ID, addr ma.Multiaddr)

	// RemovePeerAddr will unset a necessary peer.
	RemovePeerAddr(pid peer.ID)

	// RemoveAllPeer clean all necessary peers.
	RemoveAllPeer()
}
