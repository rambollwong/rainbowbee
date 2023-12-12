package network

import (
	"context"

	ma "github.com/multiformats/go-multiaddr"
)

// Dialer provides a way to establish a connection with others.
type Dialer interface {
	// Dial tries to establish an outbound connection with the remote address.
	Dial(ctx context.Context, remoteAddr ma.Multiaddr) (Connection, error)
}
