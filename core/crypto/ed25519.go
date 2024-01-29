package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/pem"
	"errors"
	"fmt"
	"io"

	"github.com/rambollwong/rainbowbee/core/crypto/pb"
)

// Ed25519PrivateKey is an ed25519 private key.
type Ed25519PrivateKey struct {
	k ed25519.PrivateKey
}

// Ed25519PublicKey is an ed25519 public key.
type Ed25519PublicKey struct {
	k ed25519.PublicKey
}

// GenerateEd25519Key generates a new ed25519 private and public key pair.
func GenerateEd25519Key(src io.Reader) (PriKey, PubKey, error) {
	pub, pri, err := ed25519.GenerateKey(src)
	if err != nil {
		return nil, nil, err
	}

	return &Ed25519PrivateKey{
			k: pri,
		},
		&Ed25519PublicKey{
			k: pub,
		},
		nil
}

// Type of the private key (Ed25519).
func (k *Ed25519PrivateKey) Type() pb.KeyType {
	return pb.KeyType_Ed25519
}

// Raw private key bytes.
func (k *Ed25519PrivateKey) Raw() ([]byte, error) {
	// The Ed25519 private key contains two 32-bytes curve points, the private
	// key and the public key.
	// It makes it more efficient to get the public key without re-computing an
	// elliptic curve multiplication.
	buf := make([]byte, len(k.k))
	copy(buf, k.k)

	return buf, nil
}

func (k *Ed25519PrivateKey) pubKeyBytes() []byte {
	return k.k[ed25519.PrivateKeySize-ed25519.PublicKeySize:]
}

// Equals compares two ed25519 private keys.
func (k *Ed25519PrivateKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PrivateKey)
	if !ok {
		return basicEquals(k, o)
	}

	return subtle.ConstantTimeCompare(k.k, edk.k) == 1
}

// GetPublic returns an ed25519 public key from a private key.
func (k *Ed25519PrivateKey) GetPublic() PubKey {
	return &Ed25519PublicKey{k: k.pubKeyBytes()}
}

// Sign returns a signature from an input message.
func (k *Ed25519PrivateKey) Sign(msg []byte) (res []byte, err error) {
	return ed25519.Sign(k.k, msg), nil
}

// Type of the public key (Ed25519).
func (k *Ed25519PublicKey) Type() pb.KeyType {
	return pb.KeyType_Ed25519
}

// Raw public key bytes.
func (k *Ed25519PublicKey) Raw() ([]byte, error) {
	return k.k, nil
}

// Equals compares two ed25519 public keys.
func (k *Ed25519PublicKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PublicKey)
	if !ok {
		return basicEquals(k, o)
	}

	return bytes.Equal(k.k, edk.k)
}

// Verify checks a signature against the input data.
func (k *Ed25519PublicKey) Verify(data []byte, sig []byte) (success bool, err error) {
	return ed25519.Verify(k.k, data, sig), nil
}

// UnmarshalEd25519PublicKey returns a public key from input bytes.
func UnmarshalEd25519PublicKey(data []byte) (PubKey, error) {
	if len(data) != 32 {
		return nil, errors.New("expect ed25519 public key data size to be 32")
	}

	return &Ed25519PublicKey{
		k: ed25519.PublicKey(data),
	}, nil
}

// UnmarshalEd25519PrivateKey returns a private key from input bytes.
func UnmarshalEd25519PrivateKey(data []byte) (PriKey, error) {
	switch len(data) {
	case ed25519.PrivateKeySize + ed25519.PublicKeySize:
		// Remove the redundant public key. See issue #36.
		redundantPk := data[ed25519.PrivateKeySize:]
		pk := data[ed25519.PrivateKeySize-ed25519.PublicKeySize : ed25519.PrivateKeySize]
		if subtle.ConstantTimeCompare(pk, redundantPk) == 0 {
			return nil, errors.New("expected redundant ed25519 public key to be redundant")
		}

		// No point in storing the extra data.
		newKey := make([]byte, ed25519.PrivateKeySize)
		copy(newKey, data[:ed25519.PrivateKeySize])
		data = newKey
	case ed25519.PrivateKeySize:
	default:
		return nil, fmt.Errorf(
			"expected ed25519 data size to be %d or %d, got %d",
			ed25519.PrivateKeySize,
			ed25519.PrivateKeySize+ed25519.PublicKeySize,
			len(data),
		)
	}

	return &Ed25519PrivateKey{
		k: ed25519.PrivateKey(data),
	}, nil
}

// PEMEncodeEd25519PrivateKey encodes an Ed25519 private key into PEM format.
func PEMEncodeEd25519PrivateKey(key *Ed25519PrivateKey) ([]byte, error) {
	raw, err := key.Raw()
	if err != nil {
		return nil, err
	}
	skPem := &pem.Block{
		Type:  PEMBlockTypeEd25519PrivateKey,
		Bytes: raw,
	}
	return pem.EncodeToMemory(skPem), nil
}

// PEMDecodeEd25519PrivateKey decodes a PEM-encoded Ed25519 private key.
func PEMDecodeEd25519PrivateKey(pemBytes []byte) (*Ed25519PrivateKey, error) {
	skPem, _ := pem.Decode(pemBytes)
	if skPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	sk, err := UnmarshalEd25519PrivateKey(skPem.Bytes)
	if err != nil {
		return nil, err
	}
	return sk.(*Ed25519PrivateKey), nil
}

// PEMEncodeEd25519PublicKey encodes an Ed25519 public key into PEM format.
func PEMEncodeEd25519PublicKey(pk *Ed25519PublicKey) ([]byte, error) {
	raw, err := pk.Raw()
	if err != nil {
		return nil, err
	}
	pkPem := &pem.Block{
		Type:  PEMBlockTypeEd25519PublicKey,
		Bytes: raw,
	}
	return pem.EncodeToMemory(pkPem), nil
}

// PEMDecodeEd25519PublicKey decodes a PEM-encoded Ed25519 public key.
func PEMDecodeEd25519PublicKey(pemBytes []byte) (*Ed25519PublicKey, error) {
	pkPem, _ := pem.Decode(pemBytes)
	if pkPem == nil {
		return nil, ErrPEMDecodeFailed
	}
	pk, err := UnmarshalEd25519PublicKey(pkPem.Bytes)
	if err != nil {
		return nil, err
	}
	return pk.(*Ed25519PublicKey), nil
}
