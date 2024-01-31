package tcp

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-yamux/v4"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/util"
)

var (
	_ network.Connection = (*Conn)(nil)

	defaultYamuxConfig = yamux.DefaultConfig()
)

func init() {
	defaultYamuxConfig.MaxStreamWindowSize = 32 << 20
	defaultYamuxConfig.LogOutput = io.Discard
	defaultYamuxConfig.ReadBufSize = 0
	defaultYamuxConfig.ConnectionWriteTimeout = 1 * time.Second
	defaultYamuxConfig.EnableKeepAlive = true
	defaultYamuxConfig.KeepAliveInterval = 10 * time.Second
}

type Conn struct {
	network.BasicStatus
	ctx context.Context

	nw *Network
	c  net.Conn

	sess *yamux.Session

	usableSSLock sync.Mutex
	usableSS     map[*yamuxReceiveStream]struct{}
	usableRSLock sync.Mutex
	usableRS     map[*yamuxSendStream]struct{}

	localAddr  ma.Multiaddr
	localPID   peer.ID
	remoteAddr ma.Multiaddr
	remotePID  peer.ID

	closeC    chan struct{}
	closeOnce sync.Once
}

func NewConn(ctx context.Context, nw *Network, c net.Conn,
	dir network.Direction, remoteAddr ma.Multiaddr) (*Conn, error) {
	conn := &Conn{
		BasicStatus:  *network.NewStatus(dir, time.Now(), nil),
		ctx:          ctx,
		nw:           nw,
		c:            nil,
		sess:         nil,
		usableSSLock: sync.Mutex{},
		usableRSLock: sync.Mutex{},
		usableSS:     make(map[*yamuxReceiveStream]struct{}),
		usableRS:     make(map[*yamuxSendStream]struct{}),
		localAddr:    nil,
		remoteAddr:   nil,
		localPID:     nw.LocalPeerID(),
		remotePID:    "",
		closeC:       make(chan struct{}),
		closeOnce:    sync.Once{},
	}
	var err error
	conn.localAddr, err = manet.FromNetAddr(c.LocalAddr())
	if err != nil {
		return nil, err
	}

	conn.remoteAddr, err = manet.FromNetAddr(c.RemoteAddr())
	if err != nil {
		return nil, err
	}
	// start handshake
	err = conn.handshakeAndAttachYamux(c, remoteAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.SetClosed()
		close(c.closeC)
		if err = c.sess.Close(); err != nil {
			return
		}
		if err = c.c.Close(); err != nil {
			return
		}
	})
	return err
}

func (c *Conn) closed() bool {
	select {
	case <-c.closeC:
		return true
	case <-c.sess.CloseChan():
		_ = c.Close()
		return true
	case <-c.ctx.Done():
		_ = c.Close()
		return true
	default:
		return false
	}
}

func (c *Conn) LocalAddr() ma.Multiaddr {
	return c.localAddr
}

func (c *Conn) LocalNetAddr() net.Addr {
	return c.c.LocalAddr()
}

func (c *Conn) LocalPeerID() peer.ID {
	return c.localPID
}

func (c *Conn) RemoteAddr() ma.Multiaddr {
	return c.remoteAddr
}

func (c *Conn) RemoteNetAddr() net.Addr {
	return c.c.RemoteAddr()
}

func (c *Conn) RemotePeerID() peer.ID {
	return c.remotePID
}

func (c *Conn) Network() network.Network {
	return c.nw
}

func (c *Conn) OpenSendStream() (network.SendStream, error) {
	if c.closed() {
		return nil, ErrConnClosed
	}
	return openSendStream(c)
}

func (c *Conn) AcceptReceiveStream() (network.ReceiveStream, error) {
	if c.closed() {
		return nil, ErrConnClosed
	}
	return acceptReceiveStream(c)
}

// handshakeInbound handshake with remote peer over new inbound connection
func (c *Conn) handshakeInbound(conn net.Conn) (net.Conn, error) {
	var err error
	finalConn := conn
	if c.nw.tlsEnabled {
		// tls handshake
		// inbound conn as server
		tlsCfg := c.nw.tlsCfg.Clone()
		tlsConn := tls.Server(finalConn, tlsCfg)
		err = tlsConn.Handshake()
		if err != nil {
			_ = tlsConn.Close()
			return nil, err
		}
		connState := tlsConn.ConnectionState()
		if connState.NegotiatedProtocol != tlsCfg.NextProtos[0] {
			return nil, ErrNextProtoMismatch
		}
		c.remotePID, err = c.nw.pidLoader(connState.PeerCertificates)
		if err != nil {
			_ = tlsConn.Close()
			return nil, err
		}
		finalConn = tlsConn
	} else {
		// exchange PID
		// receive pid
		remotePIDBz := make([]byte, peer.IDLength)
		_, err = finalConn.Read(remotePIDBz)
		if err != nil {
			_ = finalConn.Close()
			return nil, err
		}
		c.remotePID = peer.ID(remotePIDBz)
		// send pid
		_, err = finalConn.Write([]byte(c.localPID))
		if err != nil {
			_ = finalConn.Close()
			return nil, err
		}
	}
	return finalConn, nil
}

// handshakeOutbound handshake with remote peer over outbound connection
func (c *Conn) handshakeOutbound(conn net.Conn, remoteAddr ma.Multiaddr) (net.Conn, error) {
	var err error
	finalConn := conn
	if c.nw.tlsEnabled {
		// tls handshake
		// outbound conn as client
		tlsCfg := c.nw.tlsCfg.Clone()
		if remoteAddr != nil && util.ContainsDNS(remoteAddr) {
			dnsDomain, _ := ma.SplitFirst(remoteAddr)
			if dnsDomain != nil {
				tlsCfg.ServerName, _ = dnsDomain.ValueForProtocol(dnsDomain.Protocol().Code)
			}
		}

		tlsConn := tls.Client(finalConn, tlsCfg)
		err = tlsConn.Handshake()
		if err != nil {
			_ = tlsConn.Close()
			return nil, err
		}
		connState := tlsConn.ConnectionState()
		if connState.NegotiatedProtocol != tlsCfg.NextProtos[0] {
			return nil, ErrNextProtoMismatch
		}
		c.remotePID, err = c.nw.pidLoader(connState.PeerCertificates)
		if err != nil {
			_ = tlsConn.Close()
			return nil, err
		}
		finalConn = tlsConn
	} else {
		// exchange PID
		// send pid
		_, err = finalConn.Write([]byte(c.localPID))
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		// receive pid
		remotePIDBz := make([]byte, peer.IDLength)
		_, err = finalConn.Read(remotePIDBz)
		if err != nil {
			_ = finalConn.Close()
			return nil, err
		}
		c.remotePID = peer.ID(remotePIDBz)
	}
	return finalConn, nil
}

// attachYamuxInbound create sessions object that communicates with the remote peer through the inbound connection.
func (c *Conn) attachYamuxInbound(conn net.Conn) error {
	// inbound conn as server
	sess, err := yamux.Server(conn, defaultYamuxConfig, nil)
	if err != nil {
		_ = conn.Close()
		return err
	}

	c.c = conn
	c.sess = sess
	return nil
}

// attachYamuxOutbound create sessions object that communicates with the remote peer through the outbound connection
func (c *Conn) attachYamuxOutbound(conn net.Conn) error {
	// outbound conn as client
	sess, err := yamux.Client(conn, defaultYamuxConfig, nil)
	if err != nil {
		_ = conn.Close()
		return err
	}

	c.c = conn
	c.sess = sess
	return nil
}

// handshakeAndAttachYamux Process the connection object, perform the TLS handshake and create a session object
// that interacts with the renmote peer through the connection object.
func (c *Conn) handshakeAndAttachYamux(conn net.Conn, remoteAddr ma.Multiaddr) error {
	var err error
	var finalConn net.Conn
	switch c.Direction() {
	case network.Inbound:
		finalConn, err = c.handshakeInbound(conn)
		if err != nil {
			return err
		}
		err = c.attachYamuxInbound(finalConn)
		if err != nil {
			return err
		}
	case network.Outbound:
		finalConn, err = c.handshakeOutbound(conn, remoteAddr)
		if err != nil {
			return err
		}
		err = c.attachYamuxOutbound(finalConn)
		if err != nil {
			return err
		}
	default:
		_ = c.Close()
		return ErrUnknownDirection
	}
	return nil
}

func (c *Conn) putUsableSendStream(ss *yamuxReceiveStream) {
	c.usableSSLock.Lock()
	defer c.usableSSLock.Unlock()
	c.usableSS[ss] = struct{}{}
}

func (c *Conn) getUsableSendStream() (ss *yamuxReceiveStream, ok bool) {
	c.usableSSLock.Lock()
	defer c.usableSSLock.Unlock()
	var closed []*yamuxReceiveStream
	for s := range c.usableSS {
		s := s
		if s.Closed() {
			_ = s.yamuxStream.ys.CloseWrite()
			closed = append(closed, s)
			continue
		}
		ss = s
		ok = true
		break
	}
	for _, stream := range closed {
		delete(c.usableSS, stream)
	}
	delete(c.usableSS, ss)
	return ss, ok
}

func (c *Conn) putUsableReceiveStream(rs *yamuxSendStream) {
	c.usableRSLock.Lock()
	defer c.usableRSLock.Unlock()
	c.usableRS[rs] = struct{}{}
}

func (c *Conn) getUsableReceiveStream() (rs *yamuxSendStream, ok bool) {
	c.usableRSLock.Lock()
	defer c.usableRSLock.Unlock()
	var closed []*yamuxSendStream
	for s := range c.usableRS {
		s := s
		if s.Closed() {
			_ = s.yamuxStream.ys.CloseRead()
			closed = append(closed, s)
			continue
		}
		rs = s
		ok = true
		break
	}
	for _, stream := range closed {
		delete(c.usableRS, stream)
	}
	delete(c.usableRS, rs)
	return rs, ok
}
