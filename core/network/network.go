package network

import (
	"io"

	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowlog"
)

// Network is a state machine interface that provides a Dialer and a Listener to build a network.
type Network interface {
	Dialer
	Listener
	io.Closer

	// AcceptedConnChan returns a channel for notifying about new connections.
	// Whenever a new connection is accepted or dialed, Network should write the new connection to chan.
	AcceptedConnChan() <-chan Connection

	// Disconnect closes a connection.
	Disconnect(conn Connection) error

	// Closed returns whether the network is closed.
	Closed() bool

	// LocalPeerID returns the local peer ID.
	LocalPeerID() peer.ID

	// Logger returns the logger instance.
	Logger() *rainbowlog.Logger
}
