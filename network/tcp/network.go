package tcp

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"sync"

	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/reuse"
	"github.com/rambollwong/rainbowbee/core/safe"
	"github.com/rambollwong/rainbowbee/util"
	catutil "github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"
)

const (
	// NetworkVersion is the current version of tcp network.
	NetworkVersion = "v0.0.1"

	loggerLabel = "NETWORK-TCP"
)

var (
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

	tlsCfg              *tls.Config
	pidLoader           peer.IDLoader
	tlsEnabled          bool
	dialLoopbackEnabled bool
	connC               chan network.Connection

	localPID     peer.ID
	tcpListeners []manet.Listener
	listening    bool

	closeC chan struct{}

	logger *rainbowlog.Logger
}

func NewNetwork(opt ...Option) (*Network, error) {
	n := &Network{
		mu:           sync.RWMutex{},
		once:         sync.Once{},
		ctx:          context.Background(),
		tlsCfg:       nil,
		pidLoader:    nil,
		tlsEnabled:   true,
		connC:        make(chan network.Connection),
		localPID:     "",
		tcpListeners: make([]manet.Listener, 0, 2),
		listening:    false,
		closeC:       make(chan struct{}),
		logger:       nil,
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
	if n.pidLoader == nil {
		return ErrNilPIDLoader
	}
	if n.tlsCfg.Certificates == nil || len(n.tlsCfg.Certificates) == 0 {
		return ErrEmptyTlsCerts
	}
	n.tlsCfg.NextProtos = []string{"rainbow-bee-network-tcp-" + NetworkVersion}
	return nil
}

func canDial(addr ma.Multiaddr) bool {
	return dialMatcherNoP2p.Matches(addr) || dialMatcherWithP2p.Matches(addr)
}

func (n *Network) dial(ctx context.Context, remoteAddr ma.Multiaddr) (conn *Conn, err error) {
	dialRemoteAddr, remotePID := util.SplitAddrToTransportAndPID(remoteAddr)
	if dialRemoteAddr == nil && remotePID == "" || dialRemoteAddr == nil || !canDial(dialRemoteAddr) {
		return nil, ErrWrongTcpAddr
	}

	if manet.IsIPUnspecified(dialRemoteAddr) {
		return nil, ErrCanNotDialToUnspecifiedAddr
	}
	if !n.dialLoopbackEnabled && manet.IsIPLoopback(dialRemoteAddr) {
		return nil, ErrCanNotDialToLoopbackAddr
	}

	if ctx == nil {
		ctx = n.ctx
	}
	// try using each local address to dial to the remote
	for _, listener := range n.tcpListeners {
		listener := listener
		localAddr := listener.Multiaddr()
		if manet.IsIPUnspecified(localAddr) || (!n.dialLoopbackEnabled && manet.IsIPLoopback(localAddr)) {
			continue
		}

		// dial
		dialer := &manet.Dialer{
			Dialer: net.Dialer{
				Control: control,
			},
			LocalAddr: localAddr,
		}
		n.logger.Debug().
			Msg("trying to dial...").
			Str("local_address", localAddr.String()).
			Str("remote_address", dialRemoteAddr.String()).
			Done()
		c, err := dialer.DialContext(ctx, dialRemoteAddr)
		if err != nil {
			n.logger.Debug().
				Msg("failed to dial.").
				Str("local_address", localAddr.String()).
				Str("remote_address", dialRemoteAddr.String()).
				Err(err).
				Done()
			continue
		}
		// dial up successful, create new conn
		conn, err = NewConn(ctx, n, c, network.Outbound)
		if err != nil {
			n.logger.Error().Msg("failed to crease new connection").Err(err).Done()
			continue
		}
		if remotePID != "" && conn.remotePID != remotePID {
			_ = conn.Close()
			n.logger.Debug().
				Msg("pid mismatch, close the connection").
				Str("expected", remotePID.String()).
				Str("got", conn.remotePID.String()).
				Done()
			return nil, ErrPidMismatch
		}
		n.logger.Debug().Msg("new connection dialed").
			Str("local_address", conn.LocalAddr().String()).
			Str("remote_address", conn.RemoteAddr().String()).
			Str("remote_pid", conn.remotePID.String()).
			Done()
		return conn, err
	}
	n.logger.Debug().Msg("all dial failed").Done()
	return nil, ErrAllDialFailed
}

func (n *Network) Dial(ctx context.Context, remoteAddr ma.Multiaddr) (network.Connection, error) {
	// listener required
	if !n.listening {
		return nil, ErrListenerRequired
	}
	n.mu.RLock()
	defer n.mu.RUnlock()

	conn, err := n.dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}

	select {
	case <-n.ctx.Done():
		return nil, nil
	case <-n.closeC:
		return nil, nil
	case n.connC <- conn:
	}
	return conn, nil
}

func (n *Network) Close() error {
	close(n.closeC)
	// stop listening
	n.mu.Lock()
	defer n.mu.Unlock()
	n.listening = false
	for _, listener := range n.tcpListeners {
		_ = listener.Close()
	}
	return nil
}

// CanListen return whether address can be listened on.
func CanListen(addr ma.Multiaddr) bool {
	return listenMatcher.Matches(addr)
}

func (n *Network) listenTCP(ctx context.Context, addresses []ma.Multiaddr) ([]manet.Listener, error) {
	if len(addresses) == 0 {
		return nil, ErrEmptyListenAddress
	}
	listeners := make([]manet.Listener, 0, len(addresses))
	listenCfg := net.ListenConfig{
		Control: control,
	}
	for _, address := range addresses {
		if !CanListen(address) {
			return nil, ErrWrongTcpAddr
		}
		nw, addr, err := manet.DialArgs(address)
		if err != nil {
			return nil, err
		}
		// try to listen
		netListener, err := listenCfg.Listen(ctx, nw, addr)
		if err != nil {
			n.logger.Warn().
				Msg("failed to listen on address").
				Str("address", addr).
				Err(err).
				Done()
			continue
		}
		listener, err := manet.WrapNetListener(netListener)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
		n.logger.Info().
			Msg("listening...").
			Str("on", listener.Multiaddr().String()).
			Done()
	}
	return listeners, nil
}

// resetCheck reset close channel and once object
func (n *Network) resetCheck() {
	select {
	case <-n.closeC:
		n.closeC = make(chan struct{})
		n.once = sync.Once{}
	default:
	}
}

// listenerAcceptLoop this is an ongoing background task
// that receives new connections from remote peers and processes them.
func (n *Network) listenerAcceptLoop(listener manet.Listener) {
Loop:
	for {
		select {
		case <-n.ctx.Done():
			break Loop
		case <-n.closeC:
			break Loop
		default:

		}
		// block waiting for a new connection to come
		c, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed network connection") {
				break Loop
			}
			n.logger.Error().Msg("listener accept err").Err(err).Done()
			continue
		}

		// handle new connection
		conn, err := NewConn(n.ctx, n, c, network.Inbound)
		if err != nil {
			n.logger.Error().Msg("failed to crease new connection").Err(err).Done()
			continue
		}
		n.logger.Info().Msg("new connection accepted").
			Str("local_address", conn.LocalAddr().String()).
			Str("remote_address", conn.RemoteAddr().String()).
			Str("remote_pid", conn.remotePID.String()).
			Done()

		select {
		case <-n.ctx.Done():
			break Loop
		case <-n.closeC:
			break Loop
		case n.connC <- conn:
		}
	}
}

// Listen will run a task that starts creating a listener with the given address
// and waits for incoming connections to be accepted.
func (n *Network) Listen(ctx context.Context, addresses ...ma.Multiaddr) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.resetCheck()
	var err error

	if ctx == nil {
		ctx = n.ctx
	}

	n.once.Do(func() {
		n.logger.Info().Str("local_pid", n.localPID.String()).Done()
		if n.tlsEnabled {
			n.logger.Info().Str("tls", "enabled").Done()
		} else {
			n.logger.Info().Str("tls", "disabled").Done()
		}

		listenAddresses, e := manet.ResolveUnspecifiedAddresses(addresses, nil)
		if e != nil {
			err = e
			return
		}

		tcpListeners, e := n.listenTCP(ctx, listenAddresses)
		if e != nil {
			err = e
			return
		}
		if len(tcpListeners) == 0 {
			err = ErrListenerRequired
			return
		}

		for _, listener := range tcpListeners {
			listener := listener
			n.tcpListeners = append(n.tcpListeners, listener)
			safe.LoggerGo(n.logger, func() {
				n.listenerAcceptLoop(listener)
			})
		}

		n.listening = true
	})

	return err
}

// ListenAddresses return the list of the local addresses for listeners.
func (n *Network) ListenAddresses() []ma.Multiaddr {
	n.mu.Lock()
	defer n.mu.Unlock()
	res := catutil.SliceTransformType(n.tcpListeners, func(_ int, item manet.Listener) ma.Multiaddr {
		return item.Multiaddr()
	})
	return res
}

// AcceptedConnChan returns a channel for notifying about new connections.
func (n *Network) AcceptedConnChan() <-chan network.Connection {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connC
}

// Disconnect a connection.
func (n *Network) Disconnect(conn network.Connection) error {
	if n != conn.Network().(*Network) {
		return ErrNotTheSameNetwork
	}
	return conn.Close()
}

func (n *Network) Closed() bool {
	if n.closeC == nil {
		return false
	}
	select {
	case <-n.closeC:
		return true
	case <-n.ctx.Done():
		return true
	default:
		return false
	}
}

func (n *Network) LocalPeerID() peer.ID {
	return n.localPID
}

func (n *Network) Logger() *rainbowlog.Logger {
	return n.logger
}
