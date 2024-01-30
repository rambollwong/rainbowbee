package rainbowbee

import (
	"context"
	"crypto/tls"
	"errors"

	cc "github.com/rambollwong/rainbowbee/core/crypto"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/network/tcp"
	"github.com/rambollwong/rainbowlog"
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
	PrivateKey cc.PriKey
	TLSEnabled bool
	TLSConfig  *tls.Config
	PIDLoader  peer.IDLoader

	localPID peer.ID
}

// NewNetwork creates a new network instance based on the configuration.
func (c NetworkConfig) NewNetwork(logger *rainbowlog.Logger) (network.Network, error) {
	var err error
	c.localPID, err = peer.IDFromPriKey(c.PrivateKey)
	if err != nil {
		return nil, err
	}
	switch c.Type {
	case NetworkTypeTCP:
		return c.newTCPNetwork(logger)
	case NetworkTypeUDPQuic:
		return c.newQUICNetwork()
	default:
		return nil, ErrUnknownNetworkType
	}
}

// newTCPNetwork creates a new TCP network instance based on the configuration.
func (c NetworkConfig) newTCPNetwork(logger *rainbowlog.Logger) (network.Network, error) {
	var opts []tcp.Option
	opts = append(opts, tcp.WithLogger(logger))
	if c.Ctx != nil {
		opts = append(opts, tcp.WithContext(c.Ctx))
	}
	if c.TLSEnabled {
		if c.TLSConfig == nil {
			return nil, ErrNilTLSConfig
		}
		opts = append(opts, tcp.WithTLSConfig(c.TLSConfig))
	} else {
		opts = append(opts, tcp.WithNoTLS())
	}
	if len(c.localPID) > 0 {
		opts = append(opts, tcp.WithLocalPID(c.localPID))
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
