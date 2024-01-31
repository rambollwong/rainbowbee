package tcp

import (
	"sync"
	"time"

	"github.com/libp2p/go-yamux/v4"
	"github.com/rambollwong/rainbowbee/core/network"
)

// Ensure that the yamuxSendStream, yamuxReceiveStream types
// implement the respective stream interfaces.
var (
	_ network.SendStream    = (*yamuxSendStream)(nil)
	_ network.ReceiveStream = (*yamuxReceiveStream)(nil)
)

// yamuxStream represents a yamux stream.
type yamuxStream struct {
	c  *Conn
	ys *yamux.Stream

	closeOnce sync.Once
}

// Close closes the send stream.
func (y *yamuxStream) Close() error {
	var err error
	y.closeOnce.Do(func() {
		err = y.ys.Close()
	})
	return err
}

// Conn returns the network connection associated with the send stream.
func (y *yamuxStream) Conn() network.Connection {
	return y.c
}

// Write writes data to the send stream.
func (y *yamuxStream) Write(p []byte) (n int, err error) {
	return y.ys.Write(p)
}

// Read reads data from the reception stream.
func (y *yamuxStream) Read(p []byte) (n int, err error) {
	return y.ys.Read(p)
}

// yamuxSendStream represents a yamux send stream.
type yamuxSendStream struct {
	*yamuxStream
	*network.BasicStatus

	closeOnce sync.Once
}

// Close closes the send stream.
func (y *yamuxSendStream) Close() error {
	var err error
	y.closeOnce.Do(func() {
		y.SetClosed()
		err = y.ys.CloseWrite()
	})
	return err
}

// yamuxReceiveStream represents a yamux receive stream.
type yamuxReceiveStream struct {
	*yamuxStream
	*network.BasicStatus

	closeOnce sync.Once
}

// Close closes the reception stream.
func (y *yamuxReceiveStream) Close() error {
	var err error
	y.closeOnce.Do(func() {
		y.SetClosed()
		err = y.ys.CloseRead()
	})
	return err
}

// openSendStream try to open a send stream.
func openSendStream(c *Conn) (*yamuxSendStream, error) {
	ss, ok := c.getUsableSendStream()
	if ok {
		return &yamuxSendStream{
			yamuxStream: ss.yamuxStream,
			BasicStatus: network.NewStatus(network.Outbound, time.Now(), nil),
			closeOnce:   sync.Once{},
		}, nil
	}

	ys, err := c.sess.OpenStream(c.ctx)
	if err != nil {
		return nil, err
	}

	yss := &yamuxSendStream{
		yamuxStream: &yamuxStream{
			c:         c,
			ys:        ys,
			closeOnce: sync.Once{},
		},
		BasicStatus: network.NewStatus(network.Outbound, time.Now(), nil),
		closeOnce:   sync.Once{},
	}
	c.putUsableReceiveStream(yss)

	return yss, nil
}

// acceptReceiveStream try to accept a reception stream.
func acceptReceiveStream(c *Conn) (*yamuxReceiveStream, error) {
	rs, ok := c.getUsableReceiveStream()
	if ok {
		return &yamuxReceiveStream{
			yamuxStream: rs.yamuxStream,
			BasicStatus: network.NewStatus(network.Inbound, time.Now(), nil),
			closeOnce:   sync.Once{},
		}, nil
	}

	ys, err := c.sess.AcceptStream()
	if err != nil {
		return nil, err
	}

	yrs := &yamuxReceiveStream{
		yamuxStream: &yamuxStream{
			c:         c,
			ys:        ys,
			closeOnce: sync.Once{},
		},
		BasicStatus: network.NewStatus(network.Inbound, time.Now(), nil),
		closeOnce:   sync.Once{},
	}
	c.putUsableSendStream(yrs)

	return yrs, nil
}
