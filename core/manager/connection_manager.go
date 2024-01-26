package manager

import (
	"io"

	"github.com/rambollwong/rainbowbee/core"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

type ConnectionManager interface {
	host.Components
	io.Closer

	// AddPeerConnection adds a connection to the peer identified by pid.
	// Returns true if the connection was successfully added.
	AddPeerConnection(pid peer.ID, conn network.Connection) bool

	// RemovePeerConnection removes a connection from the peer identified by pid.
	// Returns true if the connection was found and removed.
	RemovePeerConnection(pid peer.ID, conn network.Connection) bool

	// ExistPeerConnection checks if a connection exists for the peer identified by pid.
	// Returns true if the connection exists.
	ExistPeerConnection(pid peer.ID, conn network.Connection) bool

	// PeerConnection returns the connection associated with the peer identified by pid.
	PeerConnection(pid peer.ID) network.Connection

	// PeerAllConnection returns all connections associated with the peer identified by pid.
	PeerAllConnection(pid peer.ID) []network.Connection

	// Connected checks if the peer identified by pid is connected.
	// Returns true if the peer is connected.
	Connected(pid peer.ID) bool

	// Allowed checks if the peer identified by pid is allowed to establish connections.
	// Returns true if the peer is allowed.
	Allowed(pid peer.ID) bool

	// MaxCountOfPeersAllowed returns the maximum number of peers allowed to establish connections.
	MaxCountOfPeersAllowed() int

	// CountOfPeers returns the current count of connected peers.
	CountOfPeers() int

	// AllPeers returns a slice containing all the peer IDs of connected peers.
	AllPeers() []peer.ID
}

// ConnectionSupervisor maintains the connection state of the necessary peers.
// If a necessary peer is not connected to us, supervisor will try to dial to it.
type ConnectionSupervisor interface {
	core.Switcher
	host.Components

	// SetPeerAddr will set a peer as a necessary peer and store the peer's address.
	SetPeerAddr(pid peer.ID, addr ma.Multiaddr)

	// RemovePeerAddr will unset a necessary peer.
	RemovePeerAddr(pid peer.ID)

	// RemoveAllPeer clean all necessary peers.
	RemoveAllPeer()

	// NoticePeerDisconnected tells the supervisor that a node is disconnected.
	NoticePeerDisconnected(pid peer.ID)
}
