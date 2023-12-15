package mgr

import (
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
)

// ProtocolSupportNotifyFunc is a function type used to notify whether a peer supports a protocol.
type ProtocolSupportNotifyFunc func(protocolID protocol.ProtocolID, pid peer.ID)

// ProtocolManager manages all protocols and protocol message handlers for all peers.
type ProtocolManager interface {
	// RegisterMsgPayloadHandler registers a protocol supported by us
	// and maps a handler.MsgPayloadHandler to this protocol.
	RegisterMsgPayloadHandler(protocolID protocol.ProtocolID, handler handler.MsgPayloadHandler) error

	// UnregisterMsgPayloadHandler unregisters a protocol supported by us.
	UnregisterMsgPayloadHandler(protocolID protocol.ProtocolID) error

	// IsRegistered returns whether a protocol given is supported by us.
	IsRegistered(protocolID protocol.ProtocolID) bool

	// GetHandler returns the handler.MsgPayloadHandler mapped to the protocol supported by us and the ID is the given protocolID.
	// If the protocol is not supported by us, it returns nil.
	GetHandler(protocolID protocol.ProtocolID) handler.MsgPayloadHandler

	// GetSelfSupportedProtocols returns a list of protocol.IDs that are supported by ourselves.
	GetSelfSupportedProtocols() []protocol.ProtocolID

	// IsPeerSupported returns whether the protocol is supported by the peer with the given peer.ID.
	// If the peer is not connected to us, it returns false.
	IsPeerSupported(pid peer.ID, protocolID protocol.ProtocolID) bool

	// GetPeerSupportedProtocols returns a list of protocol.IDs that are supported by the peer with the given peer.ID.
	GetPeerSupportedProtocols(pid peer.ID) []protocol.ProtocolID

	// SetPeerSupportedProtocols stores the protocols supported by the peer with the given peer.ID.
	SetPeerSupportedProtocols(pid peer.ID, protocolIDs []protocol.ProtocolID)

	// CleanPeerSupportedProtocols removes all records of protocols supported by the peer with the given peer.ID.
	CleanPeerSupportedProtocols(pid peer.ID)

	// SetProtocolSupportedNotifyFunc sets a function for notifying when a peer supports a protocol.
	SetProtocolSupportedNotifyFunc(notifyFunc ProtocolSupportNotifyFunc)

	// SetProtocolUnsupportedNotifyFunc sets a function for notifying when a peer no longer supports a protocol.
	SetProtocolUnsupportedNotifyFunc(notifyFunc ProtocolSupportNotifyFunc)
}

// ProtocolExchanger is usually used to exchange protocols supported by both peers.
type ProtocolExchanger interface {
	// ProtocolID returns the protocol.ID of the exchanger service.
	// The protocol ID will be registered in the host.RegisterMsgPayloadHandler method.
	ProtocolID() protocol.ProtocolID

	// Handle returns the message payload handler of the exchanger service.
	// It will be registered in the host.Host.RegisterMsgPayloadHandler method.
	Handle() handler.MsgPayloadHandler

	// ExchangeProtocol sends the protocols supported by us to the other peer and receives the protocols supported by the other peer.
	// This method is invoked during connection establishment.
	ExchangeProtocol(conn network.Connection) ([]protocol.ProtocolID, error)

	// PushProtocols sends the protocols supported by us to the other peer.
	// This method is invoked when a new protocol is registered.
	PushProtocols(pid peer.ID) error
}
