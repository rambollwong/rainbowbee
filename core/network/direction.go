package network

// Direction of a connection or a stream.
type Direction uint8

const (
	// Unknown is the default direction.
	Unknown Direction = iota
	// Inbound is for when the remote peer initiated a connection.
	Inbound
	// Outbound is for when the local peer initiated a connection.
	Outbound
)

func (d Direction) String() string {
	str := []string{"Unknown", "Inbound", "Outbound"}
	if d < 0 || int(d) >= len(str) {
		return "[unrecognized]"
	}
	return str[d]
}
