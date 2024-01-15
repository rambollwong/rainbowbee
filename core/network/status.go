package network

import (
	"sync/atomic"
	"time"
)

// Status is an interface for storing metadata of a Connection or a Stream.
type Status interface {
	Direction() Direction
	EstablishedTime() time.Time
	Extra() map[any]any
	SetClosed()
	IsClosed() bool
}

// BasicStatus stores metadata of a Connection or a Stream.
type BasicStatus struct {
	// direction specifies whether this is an inbound or an outbound connection.
	direction Direction
	// establishedTime is the timestamp when this connection was established.
	establishedTime time.Time
	// closed specifies whether this connection has been closed. 0 means open, 1 means closed
	closed uint32
	// extra stores other metadata of this connection.
	extra map[any]any
}

// NewStatus creates a new BasicStatus instance.
func NewStatus(direction Direction, establishedTime time.Time, extra map[any]any) *BasicStatus {
	return &BasicStatus{direction: direction, establishedTime: establishedTime, closed: 0, extra: extra}
}

// Direction returns the direction of the connection.
func (s *BasicStatus) Direction() Direction {
	return s.direction
}

// EstablishedTime returns the time when the connection was established.
func (s *BasicStatus) EstablishedTime() time.Time {
	return s.establishedTime
}

// Extra returns the extra metadata of the connection.
func (s *BasicStatus) Extra() map[interface{}]interface{} {
	return s.extra
}

// SetClosed marks the connection as closed.
func (s *BasicStatus) SetClosed() {
	atomic.StoreUint32(&s.closed, 1)
}

// IsClosed returns whether the connection is closed.
func (s *BasicStatus) IsClosed() bool {
	return atomic.LoadUint32(&s.closed) == 1
}
