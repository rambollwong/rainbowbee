package peer

import (
	"crypto/x509"

	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	cc "github.com/rambollwong/rainbowbee/core/crypto"
)

const IDLength = 46

var (
	ErrNoPubKey    = errors.New("public key is not embedded in peer ID")
	ErrEmptyPeerID = errors.New("empty peer ID")
)

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
	l := i
	r := other
	for i := 0; i < len(l) && i < len(r); i++ {
		if l[i] != r[i] {
			return l[i] > r[i]
		}
	}
	return true
}

// ExtractPubKey attempts to extract the public key from an ID.
//
// This method returns ErrNoPublicKey if the peer ID looks valid, but it can't extract
// the public key.
func (i ID) ExtractPubKey() (cc.PubKey, error) {
	hash, err := base58.Decode(i.String())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to decode peer ID")
	}
	decoded, err := multihash.Decode(hash)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to decode hash")
	}
	if decoded.Code != multihash.IDENTITY {
		return nil, ErrNoPubKey
	}
	pk, err := cc.ProtoUnmarshalPubKey(decoded.Digest)
	return pk, err
}

// Validate checks if ID is empty or not.
func (i ID) Validate() error {
	if len(i) == 0 {
		return ErrEmptyPeerID
	}

	return nil
}

// MatchesPubKey tests whether this ID was derived from the public key pk.
func (i ID) MatchesPubKey(pk cc.PubKey) bool {
	id, err := IDFromPubKey(pk)
	if err != nil {
		return false
	}
	return id == i
}

// MatchesPriKey tests whether this ID was derived from the secret key sk.
func (i ID) MatchesPriKey(sk cc.PriKey) bool {
	return i.MatchesPubKey(sk.GetPublic())
}

// IDLoader is a function can load the peer.ID
// from []*x509.Certificate exchanged during tls handshaking.
type IDLoader func([]*x509.Certificate) (ID, error)

// IDFromPubKey returns the Peer ID corresponding to the public key pk.
func IDFromPubKey(pk cc.PubKey) (ID, error) {
	bz, err := cc.ProtoMarshalPubKey(pk)
	if err != nil {
		return "", err
	}
	var alg uint64 = multihash.SHA2_256
	if len(bz) <= IDLength {
		alg = multihash.IDENTITY
	}
	hash, _ := multihash.Sum(bz, alg, -1)
	return ID(base58.Encode(hash)), nil
}

// IDFromPriKey returns the Peer ID corresponding to the secret key sk.
func IDFromPriKey(sk cc.PriKey) (ID, error) {
	return IDFromPubKey(sk.GetPublic())
}
