package broadcast

import (
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
)

// PubSub provides functions for broadcasting and subscribing messages to the network.
type PubSub interface {
	host.Components

	// AllMetadataOnlyPeers returns a list of peer.IDs that communicate with us using a metadata-only link.
	AllMetadataOnlyPeers() []peer.ID

	// Subscribe registers a sub-msg handler for handling messages listened from the given topic.
	Subscribe(topic string, msgHandler handler.SubMsgHandler)

	// Unsubscribe cancels listening to the given topic and unregisters the sub-msg handler for this topic.
	Unsubscribe(topic string)

	// Publish pushes a message to the network with the given topic.
	Publish(topic string, msg []byte)

	// ProtocolID returns the protocol.ID of the PubSub service.
	// The protocol ID will be registered in the host.RegisterMsgPayloadHandler method.
	ProtocolID() protocol.ID

	// ProtocolMsgHandler returns a function of type handler.MsgPayloadHandler.
	// It will be registered in the host.Host.RegisterMsgPayloadHandler method.
	ProtocolMsgHandler() handler.MsgPayloadHandler

	// HostNotifiee returns an implementation of the host.Notifiee interface.
	// It will be registered in the host.Host.Notify method.
	HostNotifiee() host.Notifiee

	// ID returns the local peer ID.
	ID() peer.ID

	// Stop stops the pub-sub service.
	Stop() error

	// SetBlackPeer adds a peer ID to the PubSub's blacklist.
	SetBlackPeer(pid peer.ID)

	// RemoveBlackPeer removes a peer ID from the PubSub's blacklist.
	RemoveBlackPeer(pid peer.ID)
}
