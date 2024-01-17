package host

import (
	"context"
	"crypto"

	"github.com/rambollwong/rainbowbee/core"
	"github.com/rambollwong/rainbowbee/core/blacklist"
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/store"

	ma "github.com/multiformats/go-multiaddr"
)

// PeerProtocols stores the peer.ID and the list of protocol.IDs supported by the peer.
type PeerProtocols struct {
	PID       peer.ID
	Protocols []protocol.ID
}

// Host provides network capabilities.
type Host interface {
	core.Switcher

	// Context returns the context of the host instance.
	Context() context.Context

	// PrivateKey returns the cryptographic private key.
	PrivateKey() crypto.PrivateKey

	// ID returns the local peer ID.
	ID() peer.ID

	// RegisterMsgPayloadHandler registers a handler.MsgPayloadHandler for handling
	// messages received with the specified protocolID.
	RegisterMsgPayloadHandler(protocolID protocol.ID, handler handler.MsgPayloadHandler) error

	// UnregisterMsgPayloadHandler unregisters the handler.MsgPayloadHandler for
	// the specified protocolID.
	UnregisterMsgPayloadHandler(protocolID protocol.ID) error

	// SendMsg sends a message with the specified protocolID to the receiver
	// identified by receiverPID.
	SendMsg(protocolID protocol.ID, receiverPID peer.ID, msgPayload []byte) error

	// Dial attempts to establish a connection with a peer at the given remote address.
	Dial(remoteAddr ma.Multiaddr) (network.Connection, error)

	// CheckClosedConnWithErr checks if the connection has been closed.
	// Returns true if conn.IsClosed() is true or if err indicates a closed connection,
	// otherwise returns false.
	CheckClosedConnWithErr(conn network.Connection, err error) bool

	// PeerStore returns the store.PeerStore instance associated with the host.
	PeerStore() store.PeerStore

	// ConnectionManager returns the manager.ConnectionManager instance associated with the host.
	ConnectionManager() manager.ConnectionManager

	// ProtocolManager returns the manager.ProtocolManager instance associated with the host.
	ProtocolManager() manager.ProtocolManager

	// PeerBlackList returns the blacklist.PeerBlackList instance associated with the host.
	PeerBlackList() blacklist.PeerBlackList

	// PeerProtocols returns the list of connected peers and their supported protocol.IDs.
	// If protocolIDs is nil, it returns information about all connected peers.
	// Otherwise, it returns information about connected peers that support the specified protocolIDs.
	PeerProtocols(protocolIDs []protocol.ID) ([]*PeerProtocols, error)

	// PeerSupportProtocol checks if the peer identified by pid supports the specified protocolID.
	// Returns true if the peer supports the protocol, otherwise returns false.
	PeerSupportProtocol(pid peer.ID, protocolID protocol.ID) bool

	// Notify registers a Notifiee to receive notifications from the host.
	Notify(notifiee Notifiee)

	// AddDirectPeer appends a direct peer to the host.
	AddDirectPeer(dp ma.Multiaddr) error

	// ClearDirectPeers removes all direct peers from the host.
	ClearDirectPeers()

	// LocalAddresses returns a list of network addresses that the listener is listening on.
	LocalAddresses() []ma.Multiaddr

	// Network returns the network.Network instance associated with the host.
	Network() network.Network
}
