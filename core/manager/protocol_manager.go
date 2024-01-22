package manager

import (
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
)

// ProtocolSupportNotifyFunc is a function type used to notify whether a peer supports a protocol.
type ProtocolSupportNotifyFunc func(protocolID protocol.ID, pid peer.ID)

// ProtocolManager manages protocols and protocol message handlers for peers.
type ProtocolManager interface {
	// RegisterMsgPayloadHandler registers a protocol and associates a handler.MsgPayloadHandler with it.
	// It returns an error if the registration fails.
	RegisterMsgPayloadHandler(protocolID protocol.ID, handler handler.MsgPayloadHandler) error

	// UnregisterMsgPayloadHandler unregisters a previously registered protocol.
	// It returns an error if the un-registration fails.
	UnregisterMsgPayloadHandler(protocolID protocol.ID) error

	// Registered checks if a protocol is registered and supported.
	// It returns true if the protocol is supported, and false otherwise.
	Registered(protocolID protocol.ID) bool

	// RegisteredAll returns a list of protocol.IDs that have registered.
	RegisteredAll() []protocol.ID

	// Handler returns the message payload handler associated with the registered protocol.
	// If the protocol is not registered or supported, it returns nil.
	Handler(protocolID protocol.ID) handler.MsgPayloadHandler

	// SupportedByPeer checks if a peer with the given peer.ID supports a specific protocol.
	// It returns true if the protocol is supported by the peer, and false otherwise.
	// If the peer is not connected, it returns false.
	SupportedByPeer(pid peer.ID, protocolID protocol.ID) bool

	// SupportedProtocolsOfPeer returns a list of protocol.IDs that are supported by the peer with the given peer.ID.
	// If the peer is not connected or doesn't support any protocols, it returns an empty list.
	SupportedProtocolsOfPeer(pid peer.ID) []protocol.ID

	// SetSupportedProtocolsOfPeer stores the protocols supported by the peer with the given peer.ID.
	SetSupportedProtocolsOfPeer(pid peer.ID, protocolIDs []protocol.ID) error

	// CleanPeer removes all records of protocols supported by the peer with the given peer.ID.
	CleanPeer(pid peer.ID) error

	// SetProtocolSupportedNotifyFunc sets a function to be called when a peer supports a protocol.
	// The function is called with the peer.ID of the supporting peer and the supported protocol ID.
	SetProtocolSupportedNotifyFunc(notifyFunc ProtocolSupportNotifyFunc) error

	// SetProtocolUnsupportedNotifyFunc sets a function to be called when a peer no longer supports a protocol.
	// The function is called with the peer.ID of the peer and the unsupported protocol ID.
	SetProtocolUnsupportedNotifyFunc(notifyFunc ProtocolSupportNotifyFunc) error
}

// ProtocolExchanger is responsible for exchanging protocols supported by both peers.
type ProtocolExchanger interface {
	// ProtocolID returns the protocol.ID of the exchanger service.
	// The protocol ID will be registered using the host.RegisterMsgPayloadHandler method.
	ProtocolID() protocol.ID

	// Handle returns the message payload handler of the exchanger service.
	// It will be registered using the host.Host.RegisterMsgPayloadHandler method.
	Handle() handler.MsgPayloadHandler

	// ExchangeProtocol sends the protocols supported by the local peer to the remote peer and receives the protocols supported by the remote peer.
	// This method is invoked during connection establishment.
	// It takes a network.Connection parameter representing the connection between the local and remote peer.
	// It returns a slice of protocol.IDs supported by the remote peer, or an error if the protocol exchange fails.
	ExchangeProtocol(conn network.Connection) ([]protocol.ID, error)

	// PushProtocols sends the protocols supported by the local peer to the remote peer.
	// This method is invoked when a new protocol is registered.
	// It takes a peer.ID parameter representing the remote peer ID to which the protocols will be pushed.
	// It returns an error if the protocol push fails.
	PushProtocols(pid peer.ID) error
}
