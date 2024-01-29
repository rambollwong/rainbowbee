package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/pem"
	"errors"
	"io"

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

	// PEMBlockTypeRsaPrivateKey represents the PEM block type for RSA private key.
	PEMBlockTypeRsaPrivateKey = "RSA PRIVATE KEY"
	// PEMBlockTypeEd25519PrivateKey represents the PEM block type for Ed25519 private key.
	PEMBlockTypeEd25519PrivateKey = "ED25519 PRIVATE KEY"
	// PEMBlockTypeSecp256k1PrivateKey represents the PEM block type for secp256k1 private key.
	PEMBlockTypeSecp256k1PrivateKey = "SECP256K1 PRIVATE KEY"
	// PEMBlockTypeECDSAPrivateKey represents the PEM block type for ECDSA private key.
	PEMBlockTypeECDSAPrivateKey = "ECDSA PRIVATE KEY"
	// PEMBlockTypeRsaPublicKey represents the PEM block type for RSA public key.
	PEMBlockTypeRsaPublicKey = "RSA PUBLIC KEY"
	// PEMBlockTypeEd25519PublicKey represents the PEM block type for Ed25519 public key.
	PEMBlockTypeEd25519PublicKey = "ED25519 PUBLIC KEY"
	// PEMBlockTypeSecp256k1PublicKey represents the PEM block type for secp256k1 public key.
	PEMBlockTypeSecp256k1PublicKey = "SECP256K1 PUBLIC KEY"
	// PEMBlockTypeECDSAPublicKey represents the PEM block type for ECDSA public key.
	PEMBlockTypeECDSAPublicKey = "ECDSA PUBLIC KEY"
)

var (
	// ErrBadKeyType is returned when a key is not supported.
	ErrBadKeyType = errors.New("invalid or unsupported key type")
	// ErrPEMDecodeFailed is returned when decode pem to key failed.
	ErrPEMDecodeFailed = errors.New("failed to decode pem")
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

// GenerateKeyPair generates a private and public key.
// When generating Rsa type Key, bits need to be provided.
func GenerateKeyPair(typ int, bits ...int) (PriKey, PubKey, error) {
	return GenerateKeyPairWithReader(typ, rand.Reader, bits...)
}

// GenerateKeyPairWithReader returns a keypair of the given type and bit-size.
// When generating Rsa type Key, bits need to be provided.
func GenerateKeyPairWithReader(typ int, src io.Reader, bits ...int) (PriKey, PubKey, error) {
	switch typ {
	case RSA:
		return GenerateRSAKeyPair(bits[0], src)
	case Ed25519:
		return GenerateEd25519Key(src)
	case Secp256k1:
		return GenerateSecp256k1Key(src)
	case ECDSA:
		return GenerateECDSAKeyPair(src)
	default:
		return nil, nil, ErrBadKeyType
	}
}

// PEMEncodePriKey encodes a private key into PEM format based on the key's type.
func PEMEncodePriKey(priKey PriKey) ([]byte, error) {
	switch priKey.Type() {
	case pb.KeyType_RSA:
		return PEMEncodeRsaPrivateKey(priKey.(*RsaPrivateKey))
	case pb.KeyType_Ed25519:
		return PEMEncodeEd25519PrivateKey(priKey.(*Ed25519PrivateKey))
	case pb.KeyType_Secp256k1:
		return PEMEncodeSecp256k1PrivateKey(priKey.(*Secp256k1PrivateKey))
	case pb.KeyType_ECDSA:
		return PEMEncodeECDSAPrivateKey(priKey.(*ECDSAPrivateKey))
	default:
		return nil, ErrBadKeyType
	}
}

// PEMDecodePriKey decodes a PEM-encoded private key based on the key's type.
func PEMDecodePriKey(pemBytes []byte) (PriKey, error) {
	skPem, _ := pem.Decode(pemBytes)
	if skPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	switch skPem.Type {
	case PEMBlockTypeRsaPrivateKey:
		return UnmarshalRsaPrivateKey(skPem.Bytes)
	case PEMBlockTypeEd25519PrivateKey:
		return UnmarshalEd25519PrivateKey(skPem.Bytes)
	case PEMBlockTypeSecp256k1PrivateKey:
		return UnmarshalSecp256k1PrivateKey(skPem.Bytes)
	case PEMBlockTypeECDSAPrivateKey:
		return UnmarshalECDSAPrivateKey(skPem.Bytes)
	default:
		return nil, ErrBadKeyType
	}
}

// PEMEncodePubKey encodes a public key into PEM format based on the key's type.
func PEMEncodePubKey(pubKey PubKey) ([]byte, error) {
	switch pubKey.Type() {
	case pb.KeyType_RSA:
		return PEMEncodeRsaPublicKey(pubKey.(*RsaPublicKey))
	case pb.KeyType_Ed25519:
		return PEMEncodeEd25519PublicKey(pubKey.(*Ed25519PublicKey))
	case pb.KeyType_Secp256k1:
		return PEMEncodeSecp256k1PublicKey(pubKey.(*Secp256k1PublicKey))
	case pb.KeyType_ECDSA:
		return PEMEncodeECDSAPublicKey(pubKey.(*ECDSAPublicKey))
	default:
		return nil, ErrBadKeyType
	}
}

// PEMDecodePubKey decodes a PEM-encoded public key based on the key's type.
func PEMDecodePubKey(pemBytes []byte) (PubKey, error) {
	pkPem, _ := pem.Decode(pemBytes)
	if pkPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	switch pkPem.Type {
	case PEMBlockTypeRsaPublicKey:
		return UnmarshalRsaPublicKey(pkPem.Bytes)
	case PEMBlockTypeEd25519PublicKey:
		return UnmarshalEd25519PublicKey(pkPem.Bytes)
	case PEMBlockTypeSecp256k1PublicKey:
		return UnmarshalSecp256k1PublicKey(pkPem.Bytes)
	case PEMBlockTypeECDSAPublicKey:
		return UnmarshalECDSAPublicKey(pkPem.Bytes)
	default:
		return nil, ErrBadKeyType
	}
}
