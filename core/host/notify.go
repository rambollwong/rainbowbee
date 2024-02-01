package host

import (
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
)

// Notifiee contains functions for host notification callbacks.
// Each callback implementation should not block and should return as quickly as possible
// to prevent degrading host performance.
type Notifiee interface {
	// OnPeerConnected is invoked when a new connection is established.
	OnPeerConnected(pid peer.ID)

	// OnPeerDisconnected is invoked when a connection is disconnected.
	OnPeerDisconnected(pid peer.ID)

	// OnPeerProtocolSupported is invoked when a peer supports a new protocol.
	OnPeerProtocolSupported(protocolID protocol.ID, pid peer.ID)

	// OnPeerProtocolUnsupported is invoked when a peer stops supporting a new protocol.
	OnPeerProtocolUnsupported(protocolID protocol.ID, pid peer.ID)
}

var _ Notifiee = (*NotifieeBundle)(nil)

// NotifieeBundle is a bundle implementation of the Notifiee interface.
type NotifieeBundle struct {
	OnPeerConnectedFunc           func(peer.ID)
	OnPeerDisconnectedFunc        func(peer.ID)
	OnPeerProtocolSupportedFunc   func(protocolID protocol.ID, pid peer.ID)
	OnPeerProtocolUnsupportedFunc func(protocolID protocol.ID, pid peer.ID)
}

// OnPeerConnected invokes the OnPeerConnectedFunc callback if it is set.
func (n *NotifieeBundle) OnPeerConnected(pid peer.ID) {
	if n.OnPeerConnectedFunc != nil {
		n.OnPeerConnectedFunc(pid)
	}
}

// OnPeerDisconnected invokes the OnPeerDisconnectedFunc callback if it is set.
func (n *NotifieeBundle) OnPeerDisconnected(pid peer.ID) {
	if n.OnPeerDisconnectedFunc != nil {
		n.OnPeerDisconnectedFunc(pid)
	}
}

// OnPeerProtocolSupported invokes the OnPeerProtocolSupportedFunc callback if it is set.
func (n *NotifieeBundle) OnPeerProtocolSupported(protocolID protocol.ID, pid peer.ID) {
	if n.OnPeerProtocolSupportedFunc != nil {
		n.OnPeerProtocolSupportedFunc(protocolID, pid)
	}
}

// OnPeerProtocolUnsupported invokes the OnPeerProtocolUnsupportedFunc callback if it is set.
func (n *NotifieeBundle) OnPeerProtocolUnsupported(protocolID protocol.ID, pid peer.ID) {
	if n.OnPeerProtocolUnsupportedFunc != nil {
		n.OnPeerProtocolUnsupportedFunc(protocolID, pid)
	}
}
