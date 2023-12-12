package network

import (
	"context"
	"io"

	ma "github.com/multiformats/go-multiaddr"
)

// Listener provides a way to accept connections established with others.
type Listener interface {
	io.Closer

	// Listen runs a task that creates listeners with the given addresses and waits for accepting inbound connections.
	// It returns an error if there is any issue starting the listeners.
	Listen(ctx context.Context, addresses ...ma.Multiaddr) error

	// ListenAddresses returns the list of local addresses for the listeners.
	ListenAddresses() []ma.Multiaddr
}
