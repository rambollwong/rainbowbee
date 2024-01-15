package tcp

import "errors"

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
	ErrAllDialFailed             = errors.New("all dial failed")

	// ErrConnClosed will be returned if the current connection closed.
	ErrConnClosed = errors.New("connection closed")
	// ErrUnknownDirection will be returned if the direction is unknown.
	ErrUnknownDirection = errors.New("unknown direction")
	// ErrNextProtoMismatch will be returned if next proto mismatch when tls handshaking.
	ErrNextProtoMismatch = errors.New("next proto mismatch")
)
