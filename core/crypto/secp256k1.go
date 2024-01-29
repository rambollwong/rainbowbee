package crypto

import (
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/rambollwong/rainbowbee/core/crypto/pb"
)

// Secp256k1PrivateKey is a Secp256k1 private key
type Secp256k1PrivateKey secp256k1.PrivateKey

// Secp256k1PublicKey is a Secp256k1 public key
type Secp256k1PublicKey secp256k1.PublicKey

// GenerateSecp256k1Key generates a new Secp256k1 private and public key pair
func GenerateSecp256k1Key(src io.Reader) (PriKey, PubKey, error) {
	privk, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, nil, err
	}

	k := (*Secp256k1PrivateKey)(privk)
	return k, k.GetPublic(), nil
}

// UnmarshalSecp256k1PrivateKey returns a private key from bytes
func UnmarshalSecp256k1PrivateKey(data []byte) (k PriKey, err error) {
	if len(data) != secp256k1.PrivKeyBytesLen {
		return nil, fmt.Errorf("expected secp256k1 data size to be %d", secp256k1.PrivKeyBytesLen)
	}

	priKey := secp256k1.PrivKeyFromBytes(data)
	return (*Secp256k1PrivateKey)(priKey), nil
}

// UnmarshalSecp256k1PublicKey returns a public key from bytes
func UnmarshalSecp256k1PublicKey(data []byte) (_k PubKey, err error) {
	k, err := secp256k1.ParsePubKey(data)
	if err != nil {
		return nil, err
	}

	return (*Secp256k1PublicKey)(k), nil
}

// PEMEncodeSecp256k1PrivateKey encodes an Secp256k1 private key into PEM format.
func PEMEncodeSecp256k1PrivateKey(key *Secp256k1PrivateKey) ([]byte, error) {
	bytes, err := key.Raw()
	if err != nil {
		return nil, err
	}
	skPem := &pem.Block{
		Type:  PEMBlockTypeSecp256k1PrivateKey,
		Bytes: bytes,
	}
	return pem.EncodeToMemory(skPem), nil
}

// PEMDecodeSecp256k1PrivateKey decodes a PEM-encoded Secp256k1 private key.
func PEMDecodeSecp256k1PrivateKey(pemBytes []byte) (*Secp256k1PrivateKey, error) {
	skPem, _ := pem.Decode(pemBytes)
	if skPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	sk, err := UnmarshalSecp256k1PrivateKey(skPem.Bytes)
	if err != nil {
		return nil, err
	}
	return sk.(*Secp256k1PrivateKey), nil
}

// PEMEncodeSecp256k1PublicKey encodes an Secp256k1 public key into PEM format.
func PEMEncodeSecp256k1PublicKey(pk *Secp256k1PublicKey) ([]byte, error) {
	bytes, err := pk.Raw()
	if err != nil {
		return nil, err
	}
	pkPem := &pem.Block{
		Type:  PEMBlockTypeSecp256k1PublicKey,
		Bytes: bytes,
	}
	return pem.EncodeToMemory(pkPem), nil
}

// PEMDecodeSecp256k1PublicKey decodes a PEM-encoded Secp256k1 public key.
func PEMDecodeSecp256k1PublicKey(pemBytes []byte) (*Secp256k1PublicKey, error) {
	pkPem, _ := pem.Decode(pemBytes)
	if pkPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	pk, err := UnmarshalSecp256k1PublicKey(pkPem.Bytes)
	if err != nil {
		return nil, err
	}
	return pk.(*Secp256k1PublicKey), nil
}

// Type returns the private key type
func (k *Secp256k1PrivateKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

// Raw returns the bytes of the key
func (k *Secp256k1PrivateKey) Raw() ([]byte, error) {
	return (*secp256k1.PrivateKey)(k).Serialize(), nil
}

// Equals compares two private keys
func (k *Secp256k1PrivateKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PrivateKey)
	if !ok {
		return basicEquals(k, o)
	}

	return k.GetPublic().Equals(sk.GetPublic())
}

// Sign returns a signature from input data
func (k *Secp256k1PrivateKey) Sign(data []byte) (_sig []byte, err error) {
	key := (*secp256k1.PrivateKey)(k)
	hash := sha256.Sum256(data)
	sig := ecdsa.Sign(key, hash[:])

	return sig.Serialize(), nil
}

// GetPublic returns a public key
func (k *Secp256k1PrivateKey) GetPublic() PubKey {
	return (*Secp256k1PublicKey)((*secp256k1.PrivateKey)(k).PubKey())
}

// Type returns the public key type
func (k *Secp256k1PublicKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

// Raw returns the bytes of the key
func (k *Secp256k1PublicKey) Raw() (res []byte, err error) {
	return (*secp256k1.PublicKey)(k).SerializeCompressed(), nil
}

// Equals compares two public keys
func (k *Secp256k1PublicKey) Equals(o Key) bool {
	sk, ok := o.(*Secp256k1PublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return (*secp256k1.PublicKey)(k).IsEqual((*secp256k1.PublicKey)(sk))
}

// Verify compares a signature against the input data
func (k *Secp256k1PublicKey) Verify(data []byte, sigStr []byte) (success bool, err error) {
	sig, err := ecdsa.ParseDERSignature(sigStr)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(data)
	return sig.Verify(hash[:], (*secp256k1.PublicKey)(k)), nil
}
