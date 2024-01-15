package network

import (
	"io"

	"github.com/rambollwong/rainbowbee/core/peer"
)

// ConnectionHandler is a function for handling connections.
type ConnectionHandler func(conn Connection) (bool, error)

// Network is a state machine interface that provides a Dialer and a Listener to build a network.
type Network interface {
	Dialer
	Listener
	io.Closer

	// SetConnHandler registers a ConnectionHandler to handle the established connections.
	SetConnHandler(handler ConnectionHandler)

	// Disconnect closes a connection.
	Disconnect(conn Connection) error

	// Closed returns whether the network is closed.
	Closed() bool

	// LocalPeerID returns the local peer ID.
	LocalPeerID() peer.ID
}
