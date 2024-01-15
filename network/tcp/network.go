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
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowbee/util"
	"github.com/rambollwong/rainbowlog"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"
)

const (
	// TCPNetworkVersion is the current version of tcp network.
	TCPNetworkVersion = "v0.0.1"

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
		logger:         log.Logger.SubLogger(rainbowlog.WithLabels(log.DefaultLoggerLabel, loggerLabel)),
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

func (n *Network) dial(ctx context.Context, remoteAddr ma.Multiaddr) (conn *Conn, err error) {
	dialRemoteAddr, remotePID := util.GetNetAddrAndPIDFromNormalMultiAddr(remoteAddr)
	if dialRemoteAddr == nil && remotePID == "" || dialRemoteAddr == nil || !canDial(dialRemoteAddr) {
		return nil, ErrWrongTcpAddr
	}
	if !canDial(dialRemoteAddr) {
		return nil, ErrWrongTcpAddr
	}

	var (
		remoteNetAddr net.Addr
	)
	remoteNetAddr, err = manet.ToNetAddr(dialRemoteAddr)
	if err != nil {
		n.logger.Error().Msg("failed to parse dial address to net address").Err(err).Done()
		return nil, err
	}

	if ctx == nil {
		ctx = n.ctx
	}
	// try using each local address to dial to the remote
	for _, localAddr := range n.localAddresses {
		localNetAddr, err := manet.ToNetAddr(localAddr)
		if err != nil {
			n.logger.Warn().Msg("failed to parse local address to net address").Err(err).Done()
			continue
		}
		// dial
		dialer := &net.Dialer{
			LocalAddr: localNetAddr,
			Control:   control,
		}
		n.logger.Info().
			Msg("trying to dial...").
			Str("local_net_address", localNetAddr.String()).
			Str("remote_net_address", remoteNetAddr.String()).
			Str("network", localNetAddr.Network()).
			Done()
		c, err := dialer.DialContext(ctx, remoteNetAddr.Network(), remoteNetAddr.String())
		if err != nil {
			n.logger.Info().
				Msg("failed to dial.").
				Str("local_net_address", localNetAddr.String()).
				Str("remote_net_address", remoteNetAddr.String()).
				Str("network", localNetAddr.Network()).
				Err(err).
				Done()
			continue
		}
		// dial up successful, create new conn
		conn, err = NewConn(ctx, n, c, network.Outbound, remoteAddr)
		if err != nil {
			n.logger.Error().Msg("failed to crease new connection").Err(err).Done()
			continue
		}
		if remotePID != "" && conn.remotePID != remotePID {
			_ = conn.Close()
			n.logger.Info().
				Msg("pid mismatch, close the connection").
				Str("expected", remotePID.String()).
				Str("got", conn.remotePID.String()).
				Done()
			return nil, ErrPidMismatch
		}
		n.logger.Info().Msg("new connection dialed").
			Str("local_net_address", localNetAddr.String()).
			Str("remote_net_address", remoteNetAddr.String()).
			Str("network", localNetAddr.Network()).
			Str("remote_pid", conn.remotePID.String()).
			Done()
		return conn, err
	}
	n.logger.Info().Msg("all dial failed").Done()
	return nil, ErrAllDialFailed
}

func (n *Network) callConnHandler(conn *Conn) bool {
	var accept = true
	var err error
	if n.connHandler != nil {
		accept, err = n.connHandler(conn)
		if err != nil {
			n.logger.Error().Msg("failed to call connection handler").Err(err).Done()
		}
		if !accept {
			n.logger.Info().Msg("connection rejected by handler, close the connection").
				Str("local_address", conn.localAddr.String()).
				Str("remote_address", conn.remoteAddr.String()).
				Str("remote_pid", conn.remotePID.String()).Done()
			_ = conn.Close()
		}
	}
	return accept
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

	accept := n.callConnHandler(conn)
	if !accept {
		return nil, ErrConnRejectedByConnHandler
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

// reGetListenAddresses get available addresses
func (n *Network) reGetListenAddresses(addr ma.Multiaddr) ([]ma.Multiaddr, error) {
	// convert multi addr to net addr
	tcpAddr, err := manet.ToNetAddr(addr)
	if err != nil {
		return nil, err
	}
	// check if ip is an unspecified address
	if tcpAddr.(*net.TCPAddr).IP.IsUnspecified() {
		// if unspecified
		// whether an ipv6 address
		isIp6 := strings.Contains(tcpAddr.(*net.TCPAddr).IP.String(), ":")
		// get local addresses usable
		addrList, err := util.GetLocalAddresses()
		if err != nil {
			return nil, err
		}
		if len(addrList) == 0 {
			return nil, ErrNoUsableLocalAddress
		}
		// split TCP protocol , like "/tcp/8081"
		_, lastAddr := ma.SplitFunc(addr, func(component ma.Component) bool {
			return component.Protocol().Code == ma.P_TCP
		})
		res := make([]ma.Multiaddr, 0, len(addrList))
		for _, address := range addrList {
			firstAddr, err := manet.FromNetAddr(address)
			if err != nil {
				return nil, err
			}
			// join ip protocol with TCP protocol
			// like "/ip4/127.0.0.1" + "/tcp/8081" -> "/ip4/127.0.0.1/tcp/8081"
			temp := ma.Join(firstAddr, lastAddr)
			tempTcpAddr, err := manet.ToNetAddr(temp)
			if err != nil {
				return nil, err
			}
			tempIsIp6 := strings.Contains(tempTcpAddr.(*net.TCPAddr).IP.String(), ":")
			// append it if they are the same version of ip,otherwise continue
			if (isIp6 && !tempIsIp6) || (!isIp6 && tempIsIp6) {
				continue
			}
			if CanListen(temp) {
				res = append(res, temp)
			}
		}
		if len(res) == 0 {
			return nil, ErrNoUsableLocalAddress
		}
		return res, nil
	}
	res, e := manet.FromNetAddr(tcpAddr)
	if e != nil {
		return nil, e
	}
	return []ma.Multiaddr{res}, nil
}

func (n *Network) listenTCP(ctx context.Context, addresses []ma.Multiaddr) ([]net.Listener, error) {
	if len(addresses) == 0 {
		return nil, ErrEmptyListenAddress
	}
	listeners := make([]net.Listener, 0, len(addresses))
	listenCfg := net.ListenConfig{Control: control}
	for _, address := range addresses {
		addr, _ := util.GetNetAddrAndPIDFromNormalMultiAddr(address)
		if addr == nil {
			return nil, ErrNilAddr
		}
		netAddr, err := manet.ToNetAddr(addr)
		if err != nil {
			return nil, err
		}
		// try to listen
		listener, err := listenCfg.Listen(ctx, netAddr.Network(), netAddr.String())
		if err != nil {
			n.logger.Warn().
				Msg("failed to listen on address").
				Str("address", netAddr.String()).
				Err(err).
				Done()
			continue
		}
		listeners = append(listeners, listener)
		n.logger.Info().
			Msg("listening...").
			Str("on", util.CreateMultiAddrWithPIDAndNetAddr(n.localPID, addr).String()).
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
func (n *Network) listenerAcceptLoop(listener net.Listener) {
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
		n.logger.Debug().
			Msg("listener accept a connection").
			Str("remote address", c.RemoteAddr().String()).
			Done()

		// handle new connection
		conn, err := NewConn(n.ctx, n, c, network.Inbound, nil)
		if err != nil {
			n.logger.Error().Msg("failed to crease new connection").Err(err).Done()
			continue
		}
		n.logger.Info().Msg("new connection accepted").
			Str("local_address", conn.localAddr.String()).
			Str("remote_address", conn.remoteAddr.String()).
			Str("remote_pid", conn.remotePID.String()).
			Done()
		// call conn handler
		accept := n.callConnHandler(conn)
		if !accept {
			n.logger.Info().Msg("connection rejected by handler, close the connection").
				Str("local_address", conn.localAddr.String()).
				Str("remote_address", conn.remoteAddr.String()).
				Str("remote_pid", conn.remotePID.String()).Done()
			_ = conn.Close()
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
		n.listening = true
		n.logger.Info().Str("local pid", n.localPID.String()).Done()
		if n.tlsEnabled {
			n.logger.Info().Str("tls", "enabled").Done()
		} else {
			n.logger.Info().Str("tls", "disabled").Done()
		}

		for _, address := range addresses {
			if !CanListen(address) {
				err = ErrWrongTcpAddr
				return
			}
			listenAddresses, e := n.reGetListenAddresses(address)
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
				go n.listenerAcceptLoop(listener)

				localAddress, e := manet.FromNetAddr(listener.Addr())
				if e != nil {
					err = e
					return
				}
				n.localAddresses = append(n.localAddresses, localAddress)
				n.tcpListeners = append(n.tcpListeners, listener)
			}
		}
	})

	return err
}

// ListenAddresses return the list of the local addresses for listeners.
func (n *Network) ListenAddresses() []ma.Multiaddr {
	n.mu.Lock()
	defer n.mu.Unlock()
	res := make([]ma.Multiaddr, len(n.localAddresses))
	copy(res, n.localAddresses)
	return res
}

// SetConnHandler register a ConnectionHandler to handle the connection established.
func (n *Network) SetConnHandler(handler network.ConnectionHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.connHandler = handler
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
	default:
		return false
	}
}

func (n *Network) LocalPeerID() peer.ID {
	return n.localPID
}
