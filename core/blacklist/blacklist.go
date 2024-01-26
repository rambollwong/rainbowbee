package blacklist

import (
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
)

// PeerBlackList is an interface for managing a blacklist of network addresses or peer IDs.
type PeerBlackList interface {
	// AddPeerID adds a peer ID to the blacklist.
	AddPeerID(pid peer.ID)

	// RemovePeerID removes a peer ID from the blacklist.
	// If the peer ID does not exist in the blacklist, it is a no-op.
	RemovePeerID(pid peer.ID)

	// AddIPAndPort adds a network address or IP to the blacklist.
	// The address should be in the format "192.168.1.2:9000" or "192.168.1.2" or "[::1]:9000" or "[::1]".
	AddIPAndPort(ipAndPort string)

	// RemoveIPAndPort removes a network address or IP from the blacklist.
	// If the address does not exist in the blacklist, it is a no-op.
	RemoveIPAndPort(ipAndPort string)

	// BlackConn checks if the remote peer ID or network address of the given connection is blacklisted.
	// Returns true if the connection is blacklisted, otherwise false.
	BlackConn(conn network.Connection) bool

	// BlackPID checks if a given peer ID is blacklisted.
	// Returns true if the peer ID is blacklisted, otherwise false.
	BlackPID(pid peer.ID) bool
}
