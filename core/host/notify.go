package host

import (
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
)

// HostNotifiee contains functions for host notification callbacks.
type HostNotifiee interface {
	// OnPeerConnected is invoked when a new connection is established.
	OnPeerConnected(pid peer.ID)

	// OnPeerDisconnected is invoked when a connection is disconnected.
	OnPeerDisconnected(pid peer.ID)

	// OnPeerProtocolSupported is invoked when a peer supports a new protocol.
	OnPeerProtocolSupported(protocolID protocol.ProtocolID, pid peer.ID)

	// OnPeerProtocolUnsupported is invoked when a peer stops supporting a new protocol.
	OnPeerProtocolUnsupported(protocolID protocol.ProtocolID, pid peer.ID)
}

var _ HostNotifiee = (*HostNotifieeBundle)(nil)

// HostNotifieeBundle is a bundle implementation of the HostNotifiee interface.
type HostNotifieeBundle struct {
	OnPeerConnectedFunc           func(peer.ID)
	OnPeerDisconnectedFunc        func(peer.ID)
	OnPeerProtocolSupportedFunc   func(protocolID protocol.ProtocolID, pid peer.ID)
	OnPeerProtocolUnsupportedFunc func(protocolID protocol.ProtocolID, pid peer.ID)
}

// OnPeerConnected invokes the OnPeerConnectedFunc callback if it is set.
func (n *HostNotifieeBundle) OnPeerConnected(pid peer.ID) {
	if n.OnPeerConnectedFunc != nil {
		n.OnPeerConnectedFunc(pid)
	}
}

// OnPeerDisconnected invokes the OnPeerDisconnectedFunc callback if it is set.
func (n *HostNotifieeBundle) OnPeerDisconnected(pid peer.ID) {
	if n.OnPeerDisconnectedFunc != nil {
		n.OnPeerDisconnectedFunc(pid)
	}
}

// OnPeerProtocolSupported invokes the OnPeerProtocolSupportedFunc callback if it is set.
func (n *HostNotifieeBundle) OnPeerProtocolSupported(protocolID protocol.ProtocolID, pid peer.ID) {
	if n.OnPeerProtocolSupportedFunc != nil {
		n.OnPeerProtocolSupportedFunc(protocolID, pid)
	}
}

// OnPeerProtocolUnsupported invokes the OnPeerProtocolUnsupportedFunc callback if it is set.
func (n *HostNotifieeBundle) OnPeerProtocolUnsupported(protocolID protocol.ProtocolID, pid peer.ID) {
	if n.OnPeerProtocolUnsupportedFunc != nil {
		n.OnPeerProtocolUnsupportedFunc(protocolID, pid)
	}
}
