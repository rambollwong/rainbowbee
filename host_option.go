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

// Option represents an option function for configuring a Host.
type Option func(*Host) error

// apply applies the provided options to the Host.
func (h *Host) apply(opt ...Option) error {
	for _, o := range opt {
		if err := o(h); err != nil {
			return err
		}
	}
	return nil
}

// WithContext sets the context for the Host.
func WithContext(ctx context.Context) Option {
	return func(h *Host) error {
		h.ctx = ctx
		return nil
	}
}

// WithListenAddresses sets the listen addresses for the Host.
func WithListenAddresses(addresses ...ma.Multiaddr) Option {
	return func(h *Host) error {
		h.cfg.ListenAddresses = addresses
		return nil
	}
}

// WithDirectPeer adds a direct peer with the given peer ID and address to the Host.
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
func WithBlackPIDs(pid ...peer.ID) Option {
	return func(h *Host) error {
		h.cfg.BlackPIDs = append(h.cfg.BlackPIDs, pid...)
		return nil
	}
}

// WithBlackNetAddr adds blacklisted network addresses to the Host.
func WithBlackNetAddr(netAddr ...string) Option {
	return func(h *Host) error {
		h.cfg.BlackNetAddresses = append(h.cfg.BlackNetAddresses, netAddr...)
		return nil
	}
}

// WithMsgCompressible sets the Host to compress messages.
func WithMsgCompressible() Option {
	return func(h *Host) error {
		h.cfg.CompressMsg = true
		return nil
	}
}

// WithPayloadUnmarshalConcurrency sets the concurrency of the payload unmarshaler.
func WithPayloadUnmarshalConcurrency(c uint8) Option {
	return func(h *Host) error {
		h.cfg.PayloadUnmarshalerConcurrency = c
		return nil
	}
}

// WithPayloadHandlerRouterConcurrency sets the concurrency of the payload handler router.
func WithPayloadHandlerRouterConcurrency(c uint8) Option {
	return func(h *Host) error {
		h.cfg.PayloadHandlerRouterConcurrency = c
		return nil
	}
}

// WithHandlerExecutorConcurrency sets the concurrency of the handler executor.
func WithHandlerExecutorConcurrency(puc uint8) Option {
	return func(h *Host) error {
		h.cfg.HandlerExecutorConcurrency = puc
		return nil
	}
}

// WithNetworkType sets the network type for the Host.
func WithNetworkType(networkType NetworkType) Option {
	return func(h *Host) error {
		h.nwCfg.Type = networkType
		return nil
	}
}

// WithPriKey sets the private key for the Host.
// This Option will also set the Local PID at the same time.
func WithPriKey(priKey cc.PriKey) Option {
	return func(h *Host) error {
		h.nwCfg.PrivateKey = priKey
		return nil
	}
}

// WithTLS enables TLS for the Host with the provided TLS configuration and peer ID loader.
func WithTLS(tlsConfig *tls.Config, pidLoader peer.IDLoader) Option {
	return func(h *Host) error {
		h.nwCfg.TLSConfig = tlsConfig.Clone()
		h.nwCfg.PIDLoader = pidLoader
		h.nwCfg.TLSEnabled = true
		return nil
	}
}

// WithEasyToUseTLS will perform the same logic as WithPriKey and use the given priKey to
// generate a tls.Config with a self-signed certificate and also set up a corresponding PIDLoader.
// This option facilitates quick start for users who do not require custom tls.Config.
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
func WithPeerStore(peerStore store.PeerStore) Option {
	return func(h *Host) error {
		h.store = peerStore
		return nil
	}
}

// WithConnectionSupervisor sets the connection supervisor for the Host.
func WithConnectionSupervisor(supervisor manager.ConnectionSupervisor) Option {
	return func(h *Host) error {
		h.supervisor = supervisor
		return nil
	}
}

// WithConnectionManager sets the connection manager for the Host.
func WithConnectionManager(connMgr manager.ConnectionManager) Option {
	return func(h *Host) error {
		h.connMgr = connMgr
		return nil
	}
}

// WithSendStreamPoolBuilder sets the send stream pool builder for the Host.
func WithSendStreamPoolBuilder(builder manager.SendStreamPoolBuilder) Option {
	return func(h *Host) error {
		h.sendStreamPoolBuilder = builder
		return nil
	}
}

// WithSendStreamMgr sets the send stream pool manager for the Host.
func WithSendStreamMgr(sendStreamMgr manager.SendStreamPoolManager) Option {
	return func(h *Host) error {
		h.sendStreamPoolMgr = sendStreamMgr
		return nil
	}
}

// WithReceiveStreamMgr sets the receive stream manager for the Host.
func WithReceiveStreamMgr(receiveStreamMgr manager.ReceiveStreamManager) Option {
	return func(h *Host) error {
		h.receiveStreamMgr = receiveStreamMgr
		return nil
	}
}

// WithProtocolManager sets the protocol manager for the Host.
func WithProtocolManager(protocolMgr manager.ProtocolManager) Option {
	return func(h *Host) error {
		h.protocolMgr = protocolMgr
		return nil
	}
}

// WithProtocolExchanger sets the protocol exchanger for the Host.
func WithProtocolExchanger(protocolExr manager.ProtocolExchanger) Option {
	return func(h *Host) error {
		h.protocolExr = protocolExr
		return nil
	}
}

// WithBlacklist sets the peer blacklist for the Host.
func WithBlacklist(blacklist blacklist.PeerBlackList) Option {
	return func(h *Host) error {
		h.blacklist = blacklist
		return nil
	}
}
