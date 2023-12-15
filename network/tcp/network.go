package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"

	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/reuse"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowlog"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"
)

const (
	// TCPNetworkVersion is the current version of tcp network.
	TCPNetworkVersion = "v0.0.1"
)

var (
	ErrNilTlsCfg                 = errors.New("nil tls config")
	ErrEmptyTlsCerts             = errors.New("empty tls certs")
	ErrNilAddr                   = errors.New("nil addr")
	ErrEmptyListenAddress        = errors.New("empty listen address")
	ErrListenerRequired          = errors.New("at least one listener is required")
	ErrConnRejectedByConnHandler = errors.New("connection rejected by conn handler")
	ErrNotTheSameNetwork         = errors.New("not the same network")
	ErrPidMismatch               = errors.New("pid mismatch")
	ErrNilLoadPidFunc            = errors.New("load peer id function required")
	ErrWrongTcpAddr              = errors.New("wrong tcp address format")
	ErrEmptyLocalPeerId          = errors.New("empty local peer id")
	ErrNoUsableLocalAddress      = errors.New("no usable local address found")
	ErrLocalPidNotSet            = errors.New("local peer id not set")

	listenMatcher      = mafmt.And(mafmt.IP, mafmt.Base(ma.P_TCP))
	dialMatcherNoP2p   = mafmt.TCP
	dialMatcherWithP2p = mafmt.And(mafmt.TCP, mafmt.Base(ma.P_P2P))

	control = reuse.Control
)

var _ network.Network = (*Network)(nil)

type Network struct {
	mu   sync.RWMutex
	once sync.Once
	ctx  context.Context

	tlsCfg      *tls.Config
	pidLoader   peer.IDLoader
	tlsEnabled  bool
	connHandler network.ConnectionHandler

	localPID       peer.ID
	localAddresses []ma.Multiaddr
	tcpListeners   []net.Listener
	listening      bool

	closeC chan struct{}

	logger *rainbowlog.Logger
}

func NewNetwork(opt ...Option) (*Network, error) {
	n := &Network{
		mu:             sync.RWMutex{},
		once:           sync.Once{},
		ctx:            context.Background(),
		tlsCfg:         nil,
		pidLoader:      nil,
		tlsEnabled:     true,
		connHandler:    nil,
		localPID:       "",
		localAddresses: make([]ma.Multiaddr, 0, 10),
		tcpListeners:   make([]net.Listener, 0, 2),
		listening:      false,
		closeC:         make(chan struct{}),
		logger:         log.Logger.SubLogger(rainbowlog.WithLabel("TCP-NETWORK")),
	}

	if err := n.apply(opt...); err != nil {
		return nil, err
	}

	if err := n.checkTlsCfg(); err != nil {
		return nil, err
	}

	if n.localPID == "" {
		return nil, ErrLocalPidNotSet
	}

	return n, nil
}

func (n *Network) checkTlsCfg() error {
	if !n.tlsEnabled {
		return nil
	}
	if n.tlsCfg == nil {
		return ErrNilTlsCfg
	}
	if n.tlsCfg.Certificates == nil || len(n.tlsCfg.Certificates) == 0 {
		return ErrEmptyTlsCerts
	}
	n.tlsCfg.NextProtos = []string{"rainbow-bee-network-tcp-" + TCPNetworkVersion}
	return nil
}

func canDial(addr ma.Multiaddr) bool {
	return dialMatcherNoP2p.Matches(addr) || dialMatcherWithP2p.Matches(addr)
}

func (n *Network) dial(ctx context.Context, remoteAddr ma.Multiaddr)

func (n *Network) Dial(ctx context.Context, remoteAddr ma.Multiaddr) (network.Connection, error) {
	//TODO implement me
	panic("implement me")
}

func (n *Network) Close() error {
	//TODO implement me
	panic("implement me")
}

func (n *Network) Listen(ctx context.Context, addresses ...ma.Multiaddr) error {
	//TODO implement me
	panic("implement me")
}

func (n *Network) ListenAddresses() []ma.Multiaddr {
	//TODO implement me
	panic("implement me")
}

func (n *Network) SetNewConnHandler(handler network.ConnectionHandler) {
	//TODO implement me
	panic("implement me")
}

func (n *Network) Disconnect(conn network.Connection) error {
	//TODO implement me
	panic("implement me")
}

func (n *Network) Closed() bool {
	//TODO implement me
	panic("implement me")
}

func (n *Network) LocalPeerID() peer.ID {
	//TODO implement me
	panic("implement me")
}
