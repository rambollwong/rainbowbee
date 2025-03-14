package tcp

import (
	"context"
	"crypto/tls"

	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowlog"
)

// Option represents a function that configures the Network instance.
type Option func(n *Network) error

// apply applies the provided configuration options to the Network instance.
// It returns an error if any option fails to apply.
func (n *Network) apply(opt ...Option) error {
	for _, o := range opt {
		if err := o(n); err != nil {
			return err
		}
	}
	return nil
}

// WithContext sets the context for the Network instance.
// The provided context is used for cancellation and timeout control.
func WithContext(ctx context.Context) Option {
	return func(n *Network) error {
		n.ctx = ctx
		return nil
	}
}

// WithTLSConfig sets the TLS configuration for the Network instance.
// It also enables TLS for secure communication.
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return func(n *Network) error {
		n.tlsCfg = tlsCfg
		n.tlsEnabled = true
		return nil
	}
}

// WithNoTLS disables TLS for the Network instance.
// Use this option to enforce plaintext communication.
func WithNoTLS() Option {
	return func(n *Network) error {
		n.tlsEnabled = false
		return nil
	}
}

// WithPIDLoader sets the peer ID loader for the Network instance.
// The peer ID loader is responsible for resolving peer IDs.
func WithPIDLoader(pidLoader peer.IDLoader) Option {
	return func(n *Network) error {
		n.pidLoader = pidLoader
		return nil
	}
}

// WithLocalPID sets the local peer ID for the Network instance.
// The local peer ID identifies this node in the network.
func WithLocalPID(localPID peer.ID) Option {
	return func(n *Network) error {
		n.localPID = localPID
		return nil
	}
}

// WithLogger sets the logger for the Network instance.
// The provided logger is wrapped with a sub-logger for consistent labeling.
func WithLogger(logger *rainbowlog.Logger) Option {
	return func(n *Network) error {
		n.logger = logger.SubLogger(rainbowlog.WithLabels(loggerLabel))
		return nil
	}
}

// WithDialLoopbackEnable returns an Option that enables loopback dialing for the Network.
// When this option is applied, the Network will allow dialing to loopback addresses (e.g., 127.0.0.1).
// This is useful for testing or scenarios where local communication is required.
func WithDialLoopbackEnable() Option {
	return func(n *Network) error {
		n.dialLoopbackEnabled = true
		return nil
	}
}
