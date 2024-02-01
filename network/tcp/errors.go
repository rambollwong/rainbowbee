package tcp

import "errors"

var (
	ErrNilTlsCfg                   = errors.New("nil tls config")
	ErrEmptyTlsCerts               = errors.New("empty tls certs")
	ErrNilAddr                     = errors.New("nil addr")
	ErrEmptyListenAddress          = errors.New("empty listen address")
	ErrListenerRequired            = errors.New("at least one listener is required")
	ErrNotTheSameNetwork           = errors.New("not the same network")
	ErrPidMismatch                 = errors.New("pid mismatch")
	ErrNilPIDLoader                = errors.New("pid loader required")
	ErrWrongTcpAddr                = errors.New("wrong tcp address format")
	ErrCanNotDialToUnspecifiedAddr = errors.New("can not dial to unspecified address")
	ErrCanNotDialToLoopbackAddr    = errors.New("can not dial to loopback address")
	ErrNoUsableLocalAddress        = errors.New("no usable local address found")
	ErrLocalPidNotSet              = errors.New("local peer id not set")
	ErrAllDialFailed               = errors.New("all dial failed")

	// ErrConnClosed will be returned if the current connection closed.
	ErrConnClosed = errors.New("connection closed")
	// ErrUnknownDirection will be returned if the direction is unknown.
	ErrUnknownDirection = errors.New("unknown direction")
	// ErrNextProtoMismatch will be returned if next proto mismatch when tls handshaking.
	ErrNextProtoMismatch = errors.New("next proto mismatch")
)
