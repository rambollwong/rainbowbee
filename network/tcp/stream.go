package tcp

import (
	"sync"
	"time"

	"github.com/libp2p/go-yamux/v4"
	"github.com/rambollwong/rainbowbee/core/network"
)

// Ensure that the yamuxSendStream, yamuxReceiveStream, and yamuxBidirectionalStream types
// implement the respective stream interfaces.
var (
	_ network.SendStream    = (*yamuxSendStream)(nil)
	_ network.ReceiveStream = (*yamuxReceiveStream)(nil)
	_ network.Stream        = (*yamuxBidirectionalStream)(nil)
)

// yamuxStream represents a yamux stream.
type yamuxStream struct {
	*network.BasicStatus

	c  *Conn
	ys *yamux.Stream

	closeOnce sync.Once
}

// yamuxSendStream represents a yamux send stream.
type yamuxSendStream yamuxStream

func openSendStream(c *Conn) (*yamuxSendStream, error) {
	ys, err := c.sessOneWay.OpenStream(c.ctx)
	if err != nil {
		return nil, err
	}
	_ = ys.CloseRead()
	return &yamuxSendStream{
		BasicStatus: network.NewStatus(network.Outbound, time.Now(), nil),
		c:           c,
		ys:          ys,
		closeOnce:   sync.Once{},
	}, nil
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

// Conn returns the network connection associated with the send stream.
func (y *yamuxSendStream) Conn() network.Connection {
	return y.c
}

// Write writes data to the send stream.
func (y *yamuxSendStream) Write(p []byte) (n int, err error) {
	return y.ys.Write(p)
}

// yamuxReceiveStream represents a yamux receive stream.
type yamuxReceiveStream yamuxStream

// acceptReceiveStream try to accept a reception stream.
func acceptReceiveStream(c *Conn) (*yamuxReceiveStream, error) {
	// Accept a yamux stream from the underlying session
	rs, err := c.sessOneWay.AcceptStream()
	if err != nil {
		return nil, err
	}

	// Close the write side of the stream to signal that no more data will be sent
	_ = rs.CloseWrite()

	// Create a new yamuxReceiveStream instance with the accepted stream and connection information
	return &yamuxReceiveStream{
		BasicStatus: network.NewStatus(network.Inbound, time.Now(), nil),
		c:           c,
		ys:          rs,
		closeOnce:   sync.Once{},
	}, nil
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

// Conn returns the network connection associated with the reception stream.
func (y *yamuxReceiveStream) Conn() network.Connection {
	return y.c
}

// Read reads data from the reception stream.
func (y *yamuxReceiveStream) Read(p []byte) (n int, err error) {
	return y.ys.Read(p)
}

// yamuxBidirectionalStream represents a yamux bidirectional stream.
type yamuxBidirectionalStream yamuxStream

// openBidirectionalStream opens a bidirectional yamux stream on the given connection.
// It returns a yamuxBidirectionalStream and an error.
func openBidirectionalStream(c *Conn) (*yamuxBidirectionalStream, error) {
	// Open a bidirectional yamux stream from the underlying session
	ys, err := c.sessBidirectional.OpenStream(c.ctx)
	if err != nil {
		return nil, err
	}

	// Create a new yamuxBidirectionalStream instance with the opened stream and connection information
	return &yamuxBidirectionalStream{
		BasicStatus: network.NewStatus(network.Outbound, time.Now(), nil),
		c:           c,
		ys:          ys,
		closeOnce:   sync.Once{},
	}, nil
}

// acceptBidirectionalStream accepts a bidirectional yamux stream on the given connection.
// It returns a yamuxBidirectionalStream and an error.
func acceptBidirectionalStream(c *Conn) (*yamuxBidirectionalStream, error) {
	// Accept a bidirectional yamux stream from the underlying session
	ys, err := c.sessBidirectional.AcceptStream()
	if err != nil {
		return nil, err
	}

	// Create a new yamuxBidirectionalStream instance with the accepted stream and connection information
	return &yamuxBidirectionalStream{
		BasicStatus: network.NewStatus(network.Inbound, time.Now(), nil),
		c:           c,
		ys:          ys,
		closeOnce:   sync.Once{},
	}, nil
}

// Close closes the bidirectional stream.
func (y *yamuxBidirectionalStream) Close() error {
	var err error
	y.closeOnce.Do(func() {
		y.SetClosed()
		err = y.ys.Close()
	})
	return err
}

// Conn returns the network connection associated with the bidirectional stream.
func (y *yamuxBidirectionalStream) Conn() network.Connection {
	return y.c
}

// Write writes data to the bidirectional stream.
func (y *yamuxBidirectionalStream) Write(p []byte) (n int, err error) {
	return y.ys.Write(p)
}

// Read reads data from the bidirectional stream.
func (y *yamuxBidirectionalStream) Read(p []byte) (n int, err error) {
	return y.ys.Read(p)
}
