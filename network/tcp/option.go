package tcp

import (
	"context"
	"crypto/tls"

	"github.com/rambollwong/rainbowbee/core/peer"
)

// Option represents a configuration option for the Network instance.
type Option func(n *Network) error

// apply loads configuration items for the Network instance.
func (n *Network) apply(opt ...Option) error {
	for _, o := range opt {
		if err := o(n); err != nil {
			return err
		}
	}
	return nil
}

// WithContext sets the context for the Network instance.
func WithContext(ctx context.Context) Option {
	return func(n *Network) error {
		n.ctx = ctx
		return nil
	}
}

// WithTLSConfig sets the TLS configuration for the Network instance.
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return func(n *Network) error {
		n.tlsCfg = tlsCfg
		n.tlsEnabled = true
		return nil
	}
}

// WithNoTLS disables TLS for the Network instance.
func WithNoTLS() Option {
	return func(n *Network) error {
		n.tlsEnabled = false
		return nil
	}
}

// WithPIDLoader sets the peer ID loader for the Network instance.
func WithPIDLoader(pidLoader peer.IDLoader) Option {
	return func(n *Network) error {
		n.pidLoader = pidLoader
		return nil
	}
}

// WithLocalPID sets the local peer ID for the Network instance.
func WithLocalPID(localPID peer.ID) Option {
	return func(n *Network) error {
		n.localPID = localPID
		return nil
	}
}
