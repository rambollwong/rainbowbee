package tcp

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-yamux/v4"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
)

var (
	_ network.Connection = (*Conn)(nil)

	// ErrConnClosed will be returned if the current connection closed.
	ErrConnClosed = errors.New("connection closed")
	// ErrUnknownDir will be returned if the direction is unknown.
	ErrUnknownDir = errors.New("unknown direction")
	// ErrNextProtoMismatch will be returned if next proto mismatch when tls handshaking.
	ErrNextProtoMismatch = errors.New("next proto mismatch")

	defaultYamuxConfig          = yamux.DefaultConfig()
	defaultYamuxConfigForStream = yamux.DefaultConfig()
)

func init() {
	defaultYamuxConfig.MaxStreamWindowSize = 32 << 20
	defaultYamuxConfig.LogOutput = io.Discard
	defaultYamuxConfig.ReadBufSize = 0
	defaultYamuxConfig.ConnectionWriteTimeout = 1 * time.Second
	defaultYamuxConfig.EnableKeepAlive = true
	defaultYamuxConfig.KeepAliveInterval = 10 * time.Second
	defaultYamuxConfigForStream.MaxStreamWindowSize = 16 << 20
	defaultYamuxConfigForStream.LogOutput = io.Discard
	defaultYamuxConfigForStream.ReadBufSize = 0
	defaultYamuxConfigForStream.EnableKeepAlive = false
}

type Conn struct {
	network.BasicStatus
	ctx context.Context

	nw *Network
	c  net.Conn

	sess              *yamux.Session
	sessOneWay        *yamux.Session
	sessBidirectional *yamux.Session

	localAddr  ma.Multiaddr
	localPID   peer.ID
	remoteAddr ma.Multiaddr
	remotePID  peer.ID

	closeC    chan struct{}
	closeOnce sync.Once
}

func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.SetClosed()
		close(c.closeC)
		if err = c.sessBidirectional.Close(); err != nil {
			return
		}
		if err = c.sessOneWay.Close(); err != nil {
			return
		}
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
	case <-c.sessOneWay.CloseChan():
		_ = c.Close()
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

func (c *Conn) OpenBidirectionalStream() (network.Stream, error) {
	if c.closed() {
		return nil, ErrConnClosed
	}
	return openBidirectionalStream(c)
}

func (c *Conn) AcceptBidirectionalStream() (network.Stream, error) {
	if c.closed() {
		return nil, ErrConnClosed
	}
	return acceptBidirectionalStream(c)
}
