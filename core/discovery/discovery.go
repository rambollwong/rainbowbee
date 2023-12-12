package discovery

import (
	"context"

	ma "github.com/multiformats/go-multiaddr"
)

// Option is a function for applying some value to Options.
type Option func(options *Options) error

// Options stores all custom parameter values for the discovery service.
type Options struct {
	Opts map[interface{}]interface{}
}

// Apply applies the options to Options.
func (o *Options) Apply(opts ...Option) error {
	for _, opt := range opts {
		err := opt(o)
		if err != nil {
			return err
		}
	}
	return nil
}

// Announcer provides a way to announce the services supported by us to the discovery service network.
type Announcer interface {
	// Announce announces a service.
	Announce(ctx context.Context, serviceName string, opts ...Option) error
}

// Discoverer provides a way to find peers who support the service with the given name.
type Discoverer interface {
	// FindPeers finds peers who support the service with the given name.
	// This is a persistent process, so you should call this method only once for each service name.
	// If you want to stop finding, you should call the cancel function provided in the context.
	FindPeers(ctx context.Context, serviceName string, opts ...Option) (<-chan ma.Multiaddr, error)
}

// Discovery contains an Announcer and a Discoverer.
// Discovery provides a way to tell others how to find us and also provides a way to find others.
type Discovery interface {
	Announcer
	Discoverer
}
