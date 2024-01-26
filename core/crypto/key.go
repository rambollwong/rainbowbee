package crypto

import (
	"crypto/subtle"
	"errors"

	"github.com/rambollwong/rainbowbee/core/crypto/pb"
	"google.golang.org/protobuf/proto"
)

const (
	// RSA is an enum for the supported RSA key type.
	RSA = iota
	// Ed25519 is an enum for the supported Ed25519 key type.
	Ed25519
	// Secp256k1 is an enum for the supported Secp256k1 key type.
	Secp256k1
	// ECDSA is an enum for the supported ECDSA key type.
	ECDSA
)

var (
	// ErrBadKeyType is returned when a key is not supported.
	ErrBadKeyType = errors.New("invalid or unsupported key type")
	// KeyTypes is a list of supported keys
	KeyTypes = []int{
		RSA,
		Ed25519,
		Secp256k1,
		ECDSA,
	}

	// PriKeyUnmarshalers is a map of PriKey unmarshalers by key type.
	PriKeyUnmarshalers = map[pb.KeyType]PriKeyUnmarshaler{
		pb.KeyType_RSA:       UnmarshalRsaPrivateKey,
		pb.KeyType_Ed25519:   UnmarshalEd25519PrivateKey,
		pb.KeyType_Secp256k1: UnmarshalSecp256k1PrivateKey,
		pb.KeyType_ECDSA:     UnmarshalECDSAPrivateKey,
	}

	// PubKeyUnmarshalers is a map of PubKey unmarshalers by key type.
	PubKeyUnmarshalers = map[pb.KeyType]PubKeyUnmarshaler{
		pb.KeyType_RSA:       UnmarshalRsaPublicKey,
		pb.KeyType_Ed25519:   UnmarshalEd25519PublicKey,
		pb.KeyType_Secp256k1: UnmarshalSecp256k1PublicKey,
		pb.KeyType_ECDSA:     UnmarshalECDSAPublicKey,
	}
)

// PubKeyUnmarshaler is a func that creates a PubKey from a given slice of bytes.
type PubKeyUnmarshaler func(data []byte) (PubKey, error)

// PriKeyUnmarshaler is a func that creates a PriKey from a given slice of bytes.
type PriKeyUnmarshaler func(data []byte) (PriKey, error)

// Key represents a crypto key that can be compared to another key.
type Key interface {
	// Equals checks whether two PubKeys are the same.
	Equals(Key) bool

	// Raw returns the raw bytes of the key (not wrapped).
	//
	// This function is the inverse of {Pri,Pub}KeyUnmarshaler.
	Raw() ([]byte, error)

	// Type returns the protobuf key type.
	Type() pb.KeyType
}

// PriKey represents a private key that can be used to generate a public key and sign data.
type PriKey interface {
	Key

	// Sign signs the given bytes.
	Sign([]byte) ([]byte, error)

	// GetPublic returns a public key paired with this private key.
	GetPublic() PubKey
}

// PubKey is a public key that can be used to verify data signed with the corresponding private key.
type PubKey interface {
	Key

	// Verify that 'sig' is the signed hash of 'data'.
	Verify(data []byte, sig []byte) (bool, error)
}

// PubKeyToProto converts a public key object into an un-serialized protobuf PublicKey message.
func PubKeyToProto(k PubKey) (*pb.PublicKey, error) {
	data, err := k.Raw()
	if err != nil {
		return nil, err
	}
	return &pb.PublicKey{
		Type: *k.Type().Enum(),
		Data: data,
	}, nil
}

// ProtoMarshalPubKey converts a public key object into a protobuf serialized public key.
func ProtoMarshalPubKey(k PubKey) ([]byte, error) {
	pbPubKey, err := PubKeyToProto(k)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbPubKey)
}

// PubKeyFromProto converts an un-serialized protobuf PublicKey message into its representative object.
func PubKeyFromProto(pbPubKey *pb.PublicKey) (PubKey, error) {
	um, ok := PubKeyUnmarshalers[pbPubKey.GetType()]
	if !ok {
		return nil, ErrBadKeyType
	}

	data := pbPubKey.GetData()

	pk, err := um(data)
	if err != nil {
		return nil, err
	}

	switch tpk := pk.(type) {
	case *RsaPublicKey:
		tpk.cached, _ = proto.Marshal(pbPubKey)
	}

	return pk, nil
}

// ProtoUnmarshalPubKey converts a protobuf serialized public key into its representative object.
func ProtoUnmarshalPubKey(data []byte) (PubKey, error) {
	pbPubKey := new(pb.PublicKey)
	err := proto.Unmarshal(data, pbPubKey)
	if err != nil {
		return nil, err
	}
	return PubKeyFromProto(pbPubKey)
}

// PriKeyFromProto converts an un-serialized protobuf PrivateKey message into its representative object.
func PriKeyFromProto(pbPriKey *pb.PrivateKey) (PriKey, error) {
	um, ok := PriKeyUnmarshalers[pbPriKey.GetType()]
	if !ok {
		return nil, ErrBadKeyType
	}

	data := pbPriKey.GetData()

	pk, err := um(data)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// ProtoUnmarshalPriKey converts a protobuf serialized private key into its representative object.
func ProtoUnmarshalPriKey(data []byte) (PriKey, error) {
	pbPriKey := new(pb.PrivateKey)
	err := proto.Unmarshal(data, pbPriKey)
	if err != nil {
		return nil, err
	}
	return PriKeyFromProto(pbPriKey)
}

// ProtoMarshalPriKey converts a key object into its protobuf serialized form.
func ProtoMarshalPriKey(k PriKey) ([]byte, error) {
	data, err := k.Raw()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(&pb.PrivateKey{
		Type: *k.Type().Enum(),
		Data: data,
	})
}

// KeyEqual checks whether two Keys are equivalent (have identical byte representations).
func KeyEqual(k1, k2 Key) bool {
	if k1 == k2 {
		return true
	}

	return k1.Equals(k2)
}

func basicEquals(k1, k2 Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
