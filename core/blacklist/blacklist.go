package blacklist

import (
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
)

// PeerBlackList is a blacklist implementation for net addresses or peer ids .
type PeerBlackList interface {
	// AddPeerID append a peer id to blacklist.
	AddPeerID(pid peer.ID)

	// RemovePeerID delete a peer id from blacklist. If pid not exist in blacklist, it is a no-op.
	RemovePeerID(pid peer.ID)

	// AddIPAndPort append a string contains an ip or a net.Addr string with an ip and a port to blacklist.
	// The string should be in the following format:
	// "192.168.1.2:9000" or "192.168.1.2" or "[::1]:9000" or "[::1]"
	AddIPAndPort(ipAndPort string)

	// RemoveIPAndPort delete a string contains an ip or a net.Addr string with an ip and a port from blacklist.
	// If the string not exist in blacklist, it is a no-op.
	RemoveIPAndPort(ipAndPort string)

	// IsBlacklisted check whether the remote peer id or the remote net address of the connection given is blacklisted.
	IsBlacklisted(conn network.Connection) bool
}
