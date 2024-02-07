package discovery

import (
	"context"

	ma "github.com/multiformats/go-multiaddr"
)

// Announcer provides a way to announce the services supported by us to the discovery service network.
type Announcer interface {
	// Announce announces a service.
	Announce(ctx context.Context, serviceName string) error
}

// Discoverer provides a way to find peers who support the service with the given name.
type Discoverer interface {
	// SearchPeers finds peers who support the service with the given name.
	// This is a persistent process, so you should call this method only once for each service name.
	// If you want to stop finding, you should call the cancel function provided in the context.
	SearchPeers(ctx context.Context, serviceName string) (<-chan []ma.Multiaddr, error)
}

// Discovery contains an Announcer and a Discoverer.
// Discovery provides a way to tell others how to find us and also provides a way to find others.
type Discovery interface {
	Announcer
	Discoverer
}
