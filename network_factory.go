package rainbowbee

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/network/tcp"
)

// NetworkType represents the type of network.
type NetworkType uint8

const (
	NetworkTypeUnknown NetworkType = iota
	NetworkTypeTCP
	NetworkTypeUDPQuic
)

var (
	networkTypes = []string{"UNKNOWN", "TCP", "UDP-QUIC"}

	ErrNilTLSConfig       = errors.New("nil tls config")
	ErrUnknownNetworkType = errors.New("unknown network type")
)

// String returns the string representation of the NetworkType.
func (t NetworkType) String() string {
	if t > 2 {
		return networkTypes[0]
	}
	return networkTypes[t]
}

// NetworkConfig represents the configuration for creating a network instance.
type NetworkConfig struct {
	Type       NetworkType
	Ctx        context.Context
	LocalPID   peer.ID
	TLSEnabled bool
	TLSConfig  *tls.Config
	PIDLoader  peer.IDLoader
}

// NewNetwork creates a new network instance based on the configuration.
func (c NetworkConfig) NewNetwork() (network.Network, error) {
	switch c.Type {
	case NetworkTypeTCP:
		return c.newTCPNetwork()
	case NetworkTypeUDPQuic:
		return c.newQUICNetwork()
	default:
		return nil, ErrUnknownNetworkType
	}
}

// newTCPNetwork creates a new TCP network instance based on the configuration.
func (c NetworkConfig) newTCPNetwork() (network.Network, error) {
	var opts []tcp.Option
	if c.Ctx != nil {
		opts = append(opts, tcp.WithContext(c.Ctx))
	}
	if c.TLSEnabled {
		if c.TLSConfig == nil {
			return nil, ErrNilTLSConfig
		}
		opts = append(opts, tcp.WithTLSConfig(c.TLSConfig))
	}
	if len(c.LocalPID) > 0 {
		opts = append(opts, tcp.WithLocalPID(c.LocalPID))
	}
	if c.PIDLoader != nil {
		opts = append(opts, tcp.WithPIDLoader(c.PIDLoader))
	}
	return tcp.NewNetwork(opts...)
}

func (c NetworkConfig) newQUICNetwork() (network.Network, error) {
	// TODO: Implement QUIC network creation.
	panic("quic network coming soon")
}
