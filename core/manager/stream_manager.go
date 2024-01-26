package manager

import (
	"io"

	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
)

// SendStreamPoolBuilder is a function type that builds a SendStreamPool for a given network connection.
type SendStreamPoolBuilder func(conn network.Connection) (SendStreamPool, error)

// SendStreamPool is a pool that stores send streams.
type SendStreamPool interface {
	io.Closer

	// Connection returns the connection for which the send streams are created.
	Connection() network.Connection

	// InitStreams opens a few send streams with the connection.
	InitStreams() error

	// BorrowStream gets a sending stream from the send stream queue in the pool.
	// The borrowed stream should be returned to this pool after successfully sending data,
	// or it should be dropped by invoking the DropStream method when errors are found in the sending data process.
	BorrowStream() (network.SendStream, error)

	// ReturnStream returns a sending stream borrowed from this pool.
	ReturnStream(network.SendStream) error

	// DropStream closes the sending stream and drops it.
	// This method should be invoked only when errors are found.
	DropStream(network.SendStream)

	// MaxSize returns the capacity of this pool.
	MaxSize() int

	// CurrentSize returns the current size of this pool.
	CurrentSize() int

	// IdleSize returns the current count of idle send streams.
	IdleSize() int
}

// SendStreamPoolManager manages all send stream pools.
type SendStreamPoolManager interface {
	host.Components
	// Reset the manager.
	Reset()

	// AddPeerConnSendStreamPool appends a stream pool for a connection of a peer.
	AddPeerConnSendStreamPool(pid peer.ID, conn network.Connection, streamPool SendStreamPool) error

	// RemovePeerConnAndCloseSendStreamPool removes a connection of a peer and closes the stream pool for it.
	RemovePeerConnAndCloseSendStreamPool(pid peer.ID, conn network.Connection) error

	// GetPeerBestConnSendStreamPool returns a stream pool for the best connection of a peer.
	GetPeerBestConnSendStreamPool(pid peer.ID) SendStreamPool
}

// ReceiveStreamManager manages all receive streams.
type ReceiveStreamManager interface {
	host.Components
	// Reset the manager.
	Reset()

	// SetPeerReceiveStreamMaxCount sets the maximum count allowed.
	SetPeerReceiveStreamMaxCount(max int)

	// AddPeerReceiveStream appends a receiving stream to the manager.
	AddPeerReceiveStream(pid peer.ID, conn network.Connection, stream network.ReceiveStream) error

	// RemovePeerReceiveStream removes a receiving stream from the manager.
	RemovePeerReceiveStream(pid peer.ID, conn network.Connection, stream network.ReceiveStream) error

	// GetCurrentPeerReceiveStreamCount returns the current count of receive streams
	// whose remote peer ID is the given pid.
	GetCurrentPeerReceiveStreamCount(pid peer.ID) int

	// ClosePeerReceiveStreams closes all the receiving streams whose remote
	// peer ID is the given pid and which were created by the given connection.
	ClosePeerReceiveStreams(pid peer.ID, conn network.Connection) error
}
