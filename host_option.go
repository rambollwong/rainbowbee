package rainbowbee

import (
	"context"
	"crypto/tls"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/blacklist"
	cc "github.com/rambollwong/rainbowbee/core/crypto"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/store"
	"github.com/rambollwong/rainbowbee/util"
)

// Option represents a function that configures the Host.
type Option func(*Host) error

// apply applies the provided options to the Host.
// It returns an error if any option fails to apply.
func (h *Host) apply(opt ...Option) error {
	for _, o := range opt {
		if err := o(h); err != nil {
			return err
		}
	}
	return nil
}

// WithContext sets the context for the Host.
// The provided context is used for cancellation and timeout control.
func WithContext(ctx context.Context) Option {
	return func(h *Host) error {
		h.ctx = ctx
		return nil
	}
}

// WithListenAddresses sets the listen addresses for the Host.
// These addresses are used to accept incoming connections.
func WithListenAddresses(addresses ...ma.Multiaddr) Option {
	return func(h *Host) error {
		h.cfg.ListenAddresses = addresses
		return nil
	}
}

// WithDirectPeer adds a direct peer with the given peer ID and address to the Host.
// Direct peers are used for establishing connections without discovery.
func WithDirectPeer(pid peer.ID, address ma.Multiaddr) Option {
	return func(h *Host) error {
		if h.cfg.DirectPeers == nil {
			h.cfg.DirectPeers = make(map[peer.ID]ma.Multiaddr)
		}
		h.cfg.DirectPeers[pid] = address
		return nil
	}
}

// WithBlackPIDs adds blacklisted peer IDs to the Host.
// Connections to these peers will be rejected.
func WithBlackPIDs(pid ...peer.ID) Option {
	return func(h *Host) error {
		h.cfg.BlackPIDs = append(h.cfg.BlackPIDs, pid...)
		return nil
	}
}

// WithBlackNetAddr adds blacklisted network addresses to the Host.
// Connections to these addresses will be rejected.
func WithBlackNetAddr(netAddr ...string) Option {
	return func(h *Host) error {
		h.cfg.BlackNetAddresses = append(h.cfg.BlackNetAddresses, netAddr...)
		return nil
	}
}

// WithMsgCompressible enables message compression for the Host.
// Compressed messages reduce network bandwidth usage.
func WithMsgCompressible() Option {
	return func(h *Host) error {
		h.cfg.CompressMsg = true
		return nil
	}
}

// WithPayloadUnmarshalConcurrency sets the concurrency level for the payload unmarshaler.
// Higher concurrency improves message processing throughput.
func WithPayloadUnmarshalConcurrency(c uint8) Option {
	return func(h *Host) error {
		h.cfg.PayloadUnmarshalerConcurrency = c
		return nil
	}
}

// WithPayloadHandlerRouterConcurrency sets the concurrency level for the payload handler router.
// Higher concurrency improves message routing throughput.
func WithPayloadHandlerRouterConcurrency(c uint8) Option {
	return func(h *Host) error {
		h.cfg.PayloadHandlerRouterConcurrency = c
		return nil
	}
}

// WithHandlerExecutorConcurrency sets the concurrency level for the handler executor.
// Higher concurrency improves handler execution throughput.
func WithHandlerExecutorConcurrency(puc uint8) Option {
	return func(h *Host) error {
		h.cfg.HandlerExecutorConcurrency = puc
		return nil
	}
}

// WithNetworkType sets the network type for the Host.
// The network type determines the underlying transport protocol.
func WithNetworkType(networkType NetworkType) Option {
	return func(h *Host) error {
		h.nwCfg.Type = networkType
		return nil
	}
}

// WithPriKey sets the private key for the Host.
// This also sets the local peer ID derived from the private key.
func WithPriKey(priKey cc.PriKey) Option {
	return func(h *Host) error {
		h.nwCfg.PrivateKey = priKey
		return nil
	}
}

// WithTLS enables TLS for the Host with the provided TLS configuration and peer ID loader.
// TLS ensures secure communication between peers.
func WithTLS(tlsConfig *tls.Config, pidLoader peer.IDLoader) Option {
	return func(h *Host) error {
		h.nwCfg.TLSConfig = tlsConfig.Clone()
		h.nwCfg.PIDLoader = pidLoader
		h.nwCfg.TLSEnabled = true
		return nil
	}
}

// WithEasyToUseTLS enables TLS with a self-signed certificate and a default peer ID loader.
// This option simplifies TLS setup for quick start scenarios.
func WithEasyToUseTLS(priKey cc.PriKey) Option {
	return func(h *Host) (err error) {
		// Assign the priKey to the PrivateKey field of the network configuration
		h.nwCfg.PrivateKey = priKey

		// Generate a TLS configuration using the EasyToUseTLSConfig function
		h.nwCfg.TLSConfig, err = util.EasyToUseTLSConfig(h.nwCfg.PrivateKey, nil)
		if err != nil {
			return err
		}

		// Set the PIDLoader field of the network configuration to EasyToUsePIDLoader function
		h.nwCfg.PIDLoader = util.EasyToUsePIDLoader

		// Set the TLSEnabled field of the network configuration to true
		h.nwCfg.TLSEnabled = true

		return nil
	}
}

// WithPeerStore sets the peer store for the Host.
// The peer store manages peer information and metadata.
func WithPeerStore(peerStore store.PeerStore) Option {
	return func(h *Host) error {
		h.store = peerStore
		return nil
	}
}

// WithConnectionSupervisor sets the connection supervisor for the Host.
// The connection supervisor monitors and manages active connections.
func WithConnectionSupervisor(supervisor manager.ConnectionSupervisor) Option {
	return func(h *Host) error {
		h.supervisor = supervisor
		return nil
	}
}

// WithConnectionManager sets the connection manager for the Host.
// The connection manager handles connection establishment and teardown.
func WithConnectionManager(connMgr manager.ConnectionManager) Option {
	return func(h *Host) error {
		h.connMgr = connMgr
		return nil
	}
}

// WithSendStreamPoolBuilder sets the send stream pool builder for the Host.
// The builder creates pools of send streams for efficient message transmission.
func WithSendStreamPoolBuilder(builder manager.SendStreamPoolBuilder) Option {
	return func(h *Host) error {
		h.sendStreamPoolBuilder = builder
		return nil
	}
}

// WithSendStreamMgr sets the send stream pool manager for the Host.
// The manager oversees the lifecycle of send streams.
func WithSendStreamMgr(sendStreamMgr manager.SendStreamPoolManager) Option {
	return func(h *Host) error {
		h.sendStreamPoolMgr = sendStreamMgr
		return nil
	}
}

// WithReceiveStreamMgr sets the receive stream manager for the Host.
// The receive stream manager handles incoming streams and dispatches them to the appropriate handlers.
func WithReceiveStreamMgr(receiveStreamMgr manager.ReceiveStreamManager) Option {
	return func(h *Host) error {
		h.receiveStreamMgr = receiveStreamMgr
		return nil
	}
}

// WithProtocolManager sets the protocol manager for the Host.
// The protocol manager is responsible for registering and managing supported protocols.
func WithProtocolManager(protocolMgr manager.ProtocolManager) Option {
	return func(h *Host) error {
		h.protocolMgr = protocolMgr
		return nil
	}
}

// WithProtocolExchanger sets the protocol exchanger for the Host.
// The protocol exchanger facilitates protocol negotiation and selection between peers.
func WithProtocolExchanger(protocolExr manager.ProtocolExchanger) Option {
	return func(h *Host) error {
		h.protocolExr = protocolExr
		return nil
	}
}

// WithBlacklist sets the peer blacklist for the Host.
// Blacklisted peers are prevented from connecting to or interacting with the Host.
func WithBlacklist(blacklist blacklist.PeerBlackList) Option {
	return func(h *Host) error {
		h.blacklist = blacklist
		return nil
	}
}

// WithDialLoopbackEnable enables loopback dialing for the Host.
// When enabled, the Host can establish connections to loopback addresses (e.g., 127.0.0.1).
// This is particularly useful for testing or local communication scenarios.
func WithDialLoopbackEnable() Option {
	return func(h *Host) error {
		h.nwCfg.DialLoopbackEnabled = true
		return nil
	}
}
