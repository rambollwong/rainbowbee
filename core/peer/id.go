package peer

// ID represents the unique identity of a peer.
type ID string

// ToString converts the ID to a string.
func (i ID) ToString() string {
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
