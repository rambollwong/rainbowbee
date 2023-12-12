package network

import "io"

type stream interface {
	io.Closer
	Status

	Conn() Connection
}

// SendStream is an interface that defines a way to send data.
type SendStream interface {
	stream
	io.Writer
}

// ReceiveStream is an interface that defines a way to receive data.
type ReceiveStream interface {
	stream
	io.Reader
}

// Stream is an interface that defines both ways to send and receive data.
type Stream interface {
	SendStream
	ReceiveStream
}
