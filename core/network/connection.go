package network

import (
	"io"
	"net"

	"github.com/rambollwong/rainbowbee/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// Connection defines a connection with a remote peer.
type Connection interface {
	io.Closer
	Status

	// LocalAddr returns the local net multi-address of the connection.
	LocalAddr() ma.Multiaddr

	// LocalNetAddr returns the local net address of the connection.
	LocalNetAddr() net.Addr

	// LocalPeerID returns the local peer ID of the connection.
	LocalPeerID() peer.ID

	// RemoteAddr returns the remote net multi-address of the connection.
	RemoteAddr() ma.Multiaddr

	// RemoteNetAddr returns the remote net address of the connection.
	RemoteNetAddr() net.Addr

	// RemotePeerID returns the remote peer ID of the connection.
	RemotePeerID() peer.ID

	// Network returns the network instance that created this connection.
	Network() Network

	// OpenSendStream tries to open a sending stream with the connection.
	OpenSendStream() (SendStream, error)

	// AcceptReceiveStream accepts a receiving stream with the connection.
	// It will block until a new receiving stream is accepted or the connection is closed.
	AcceptReceiveStream() (ReceiveStream, error)
}
