package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"io"
	"math/big"

	"github.com/rambollwong/rainbowbee/core/crypto/pb"
)

// ECDSAPrivateKey is an implementation of an ECDSA private key.
type ECDSAPrivateKey struct {
	pri *ecdsa.PrivateKey
}

// ECDSAPublicKey is an implementation of an ECDSA public key.
type ECDSAPublicKey struct {
	pub *ecdsa.PublicKey
}

// ECDSASig holds the r and s values of an ECDSA signature.
type ECDSASig struct {
	R, S *big.Int
}

var (
	// ErrNotECDSAPubKey is returned when the public key passed is not an ecdsa public key.
	ErrNotECDSAPubKey = errors.New("not an ecdsa public key")
	// ErrNilSig is returned when the signature is nil.
	ErrNilSig = errors.New("sig is nil")
	// ErrNilPrivateKey is returned when a nil private key is provided.
	ErrNilPrivateKey = errors.New("private key is nil")
	// ErrNilPublicKey is returned when a nil public key is provided.
	ErrNilPublicKey = errors.New("public key is nil")
	// ECDSACurve is the default ecdsa curve used.
	ECDSACurve = elliptic.P256()
)

// GenerateECDSAKeyPair generates a new ecdsa private and public key.
func GenerateECDSAKeyPair(src io.Reader) (PriKey, PubKey, error) {
	return GenerateECDSAKeyPairWithCurve(ECDSACurve, src)
}

// GenerateECDSAKeyPairWithCurve generates a new ecdsa private and public key with a specified curve.
func GenerateECDSAKeyPairWithCurve(curve elliptic.Curve, src io.Reader) (PriKey, PubKey, error) {
	pri, err := ecdsa.GenerateKey(curve, src)
	if err != nil {
		return nil, nil, err
	}

	return &ECDSAPrivateKey{pri}, &ECDSAPublicKey{&pri.PublicKey}, nil
}

// ECDSAKeyPairFromKey generates a new ecdsa private and public key from an input private key.
func ECDSAKeyPairFromKey(pri *ecdsa.PrivateKey) (PriKey, PubKey, error) {
	if pri == nil {
		return nil, nil, ErrNilPrivateKey
	}

	return &ECDSAPrivateKey{pri}, &ECDSAPublicKey{&pri.PublicKey}, nil
}

// ECDSAPublicKeyFromPubKey generates a new ecdsa public key from an input public key.
func ECDSAPublicKeyFromPubKey(pub ecdsa.PublicKey) (PubKey, error) {
	return &ECDSAPublicKey{pub: &pub}, nil
}

// MarshalECDSAPrivateKey returns x509 bytes from a private key.
func MarshalECDSAPrivateKey(ePriv ECDSAPrivateKey) (res []byte, err error) {
	return x509.MarshalECPrivateKey(ePriv.pri)
}

// MarshalECDSAPublicKey returns x509 bytes from a public key.
func MarshalECDSAPublicKey(ePub ECDSAPublicKey) (res []byte, err error) {
	return x509.MarshalPKIXPublicKey(ePub.pub)
}

// UnmarshalECDSAPrivateKey returns a private key from x509 bytes.
func UnmarshalECDSAPrivateKey(data []byte) (res PriKey, err error) {
	pri, err := x509.ParseECPrivateKey(data)
	if err != nil {
		return nil, err
	}

	return &ECDSAPrivateKey{pri}, nil
}

// UnmarshalECDSAPublicKey returns the public key from x509 bytes.
func UnmarshalECDSAPublicKey(data []byte) (key PubKey, err error) {
	pubIfc, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}

	pub, ok := pubIfc.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrNotECDSAPubKey
	}

	return &ECDSAPublicKey{pub}, nil
}

// Type returns the key type.
func (ePri *ECDSAPrivateKey) Type() pb.KeyType {
	return pb.KeyType_ECDSA
}

// Raw returns x509 bytes from a private key.
func (ePri *ECDSAPrivateKey) Raw() (res []byte, err error) {
	return x509.MarshalECPrivateKey(ePri.pri)
}

// Equals compares two private keys.
func (ePri *ECDSAPrivateKey) Equals(o Key) bool {
	return basicEquals(ePri, o)
}

// Sign returns the signature of the input data.
func (ePri *ECDSAPrivateKey) Sign(data []byte) (sig []byte, err error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, ePri.pri, hash[:])
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(ECDSASig{
		R: r,
		S: s,
	})
}

// GetPublic returns a public key.
func (ePri *ECDSAPrivateKey) GetPublic() PubKey {
	return &ECDSAPublicKey{&ePri.pri.PublicKey}
}

// Type returns the key type.
func (ePub *ECDSAPublicKey) Type() pb.KeyType {
	return pb.KeyType_ECDSA
}

// Raw returns x509 bytes from a public key.
func (ePub *ECDSAPublicKey) Raw() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(ePub.pub)
}

// Equals compares to public keys.
func (ePub *ECDSAPublicKey) Equals(o Key) bool {
	return basicEquals(ePub, o)
}

// Verify compares data to a signature.
func (ePub *ECDSAPublicKey) Verify(data, sigBytes []byte) (success bool, err error) {
	sig := new(ECDSASig)
	if _, err := asn1.Unmarshal(sigBytes, sig); err != nil {
		return false, err
	}

	hash := sha256.Sum256(data)

	return ecdsa.Verify(ePub.pub, hash[:], sig.R, sig.S), nil
}
