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

var (
	directions = []string{"Unknown", "Inbound", "Outbound"}
)

func (d Direction) String() string {
	if d < 0 || int(d) >= len(directions) {
		return "[unrecognized]"
	}
	return directions[d]
}
