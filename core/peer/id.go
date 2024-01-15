package peer

import "crypto/x509"

const IDLength = 46

// ID represents the unique identity of a peer.
type ID string

// String converts the ID to a string.
func (i ID) String() string {
	return string(i)
}

// WeightCompare compares the weight of our ID with another ID.
// It determines which peer should be saved when a conflict is found between two peers.
// The function returns true if the weight of our ID is higher than the other ID's weight.
func (i ID) WeightCompare(other ID) bool {
	l := string(i)
	r := string(other)
	for i := 0; i < len(l) && i < len(r); i++ {
		if l[i] != r[i] {
			return l[i] > r[i]
		}
	}
	return true
}

// IDLoader is a function can load the peer.ID
// from []*x509.Certificate exchanged during tls handshaking.
type IDLoader func([]*x509.Certificate) (ID, error)
