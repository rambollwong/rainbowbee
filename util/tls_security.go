package util

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"time"

	"github.com/pkg/errors"
	cc "github.com/rambollwong/rainbowbee/core/crypto"
	"github.com/rambollwong/rainbowbee/core/peer"
)

/**
NOTE: The code implementation in this file is based on (libp2p)[https://github.com/libp2p/go-libp2p/blob/master/p2p/security/tls/crypto.go].
*/

const (
	certValidityPeriod        = 100 * 365 * 24 * time.Hour // ~100 years
	certificatePrefix         = "rainbow-bee-tls-handshake:"
	alpn               string = "rainbow-bee"
)

var (
	extensionPrefix   = []int{1, 3, 6, 1, 4, 1, 53594}
	extensionID       = getPrefixedExtensionID([]int{1, 1})
	extensionCritical bool // so we can mark the extension critical in tests
)

type signedKey struct {
	PubKey    []byte
	Signature []byte
}

// getPrefixedExtensionID returns an Object Identifier that can be used in x509 Certificates.
func getPrefixedExtensionID(suffix []int) []int {
	return append(extensionPrefix, suffix...)
}

// extensionIDEqual compares two extension IDs.
func extensionIDEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// tlsCertTemplate returns the template for generating an Identity's TLS certificates.
func tlsCertTemplate() (*x509.Certificate, error) {
	bigNum := big.NewInt(1 << 62)
	sn, err := rand.Int(rand.Reader, bigNum)
	if err != nil {
		return nil, err
	}

	subjectSN, err := rand.Int(rand.Reader, bigNum)
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: sn,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(certValidityPeriod),
		// According to RFC 3280, the issuer field must be set,
		// see https://datatracker.ietf.org/doc/html/rfc3280#section-4.1.2.4.
		Subject: pkix.Name{SerialNumber: subjectSN.String()},
	}, nil
}

// GenerateSignedExtension uses the provided private key to sign the public key, and returns the
// signature within a pkix.Extension.
// This extension is included in a certificate to cryptographically tie it to the rainbow-bee private key.
func GenerateSignedExtension(sk cc.PriKey, pubKey crypto.PublicKey) (pkix.Extension, error) {
	keyBytes, err := cc.ProtoMarshalPubKey(sk.GetPublic())
	if err != nil {
		return pkix.Extension{}, err
	}
	certKeyPub, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return pkix.Extension{}, err
	}
	signature, err := sk.Sign(append([]byte(certificatePrefix), certKeyPub...))
	if err != nil {
		return pkix.Extension{}, err
	}
	value, err := asn1.Marshal(signedKey{
		PubKey:    keyBytes,
		Signature: signature,
	})
	if err != nil {
		return pkix.Extension{}, err
	}

	return pkix.Extension{Id: extensionID, Critical: extensionCritical, Value: value}, nil
}

// keyToCertificate generates a new ECDSA private key and corresponding x509 certificate.
// The certificate includes an extension that cryptographically ties it to the provided rainbow-bee
// private key to authenticate TLS connections.
func keyToCertificate(sk cc.PriKey, certTmpl *x509.Certificate) (*tls.Certificate, error) {
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// after calling CreateCertificate, these will end up in Certificate.Extensions
	extension, err := GenerateSignedExtension(sk, certKey.Public())
	if err != nil {
		return nil, err
	}
	certTmpl.ExtraExtensions = append(certTmpl.ExtraExtensions, extension)

	certDER, err := x509.CreateCertificate(rand.Reader, certTmpl, certTmpl, certKey.Public(), certKey)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  certKey,
	}, nil
}

// PubKeyFromCertChain verifies the certificate chain and extract the remote's public key.
func PubKeyFromCertChain(chain []*x509.Certificate) (cc.PubKey, error) {
	if len(chain) != 1 {
		return nil, errors.New("expected one certificates in the chain")
	}
	cert := chain[0]
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	var found bool
	var keyExt pkix.Extension
	// find the rainbow-bee key extension, skipping all unknown extensions
	for _, ext := range cert.Extensions {
		if extensionIDEqual(ext.Id, extensionID) {
			keyExt = ext
			found = true
			for i, oident := range cert.UnhandledCriticalExtensions {
				if oident.Equal(ext.Id) {
					// delete the extension from UnhandledCriticalExtensions
					cert.UnhandledCriticalExtensions = append(cert.UnhandledCriticalExtensions[:i], cert.UnhandledCriticalExtensions[i+1:]...)
					break
				}
			}
			break
		}
	}
	if !found {
		return nil, errors.New("expected certificate to contain the key extension")
	}
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		// If we return a x509 error here, it will be sent on the wire.
		// Wrap the error to avoid that.
		return nil, errors.WithMessage(err, "certificate verification failed")
	}

	var sk signedKey
	if _, err := asn1.Unmarshal(keyExt.Value, &sk); err != nil {
		return nil, errors.WithMessage(err, "unmarshalling signed certificate failed")
	}
	pubKey, err := cc.ProtoUnmarshalPubKey(sk.PubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshalling public key failed")
	}
	certKeyPub, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, err
	}
	valid, err := pubKey.Verify(append([]byte(certificatePrefix), certKeyPub...), sk.Signature)
	if err != nil {
		return nil, errors.WithMessage(err, "signature verification failed")
	}
	if !valid {
		return nil, errors.New("signature invalid")
	}
	return pubKey, nil
}

// EasyToUseTLSConfig generates a TLS configuration based on the provided private key and certificate template.
// If the certificate template is nil, it generates a new one using tlsCertTemplate().
// It returns a *tls.Config that can be used to configure TLS settings for a server or client.
func EasyToUseTLSConfig(sk cc.PriKey, certTemplate *x509.Certificate) (*tls.Config, error) {
	var err error
	if certTemplate == nil {
		certTemplate, err = tlsCertTemplate()
		if err != nil {
			return nil, err
		}
	}

	// Generate a TLS certificate using the private key and certificate template
	cert, err := keyToCertificate(sk, certTemplate)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // This is not insecure here. We will verify the cert chain ourselves.
		ClientAuth:         tls.RequireAnyClientCert,
		Certificates:       []tls.Certificate{*cert},
		VerifyPeerCertificate: func(_ [][]byte, _ [][]*x509.Certificate) error {
			// The verification of the peer certificate is not specialized in this TLS config.
			return nil
		},
		NextProtos:             []string{alpn},
		SessionTicketsDisabled: true,
	}, nil
}

// EasyToUsePIDLoader generates a peer ID based on the provided certificate chain.
// It extracts the public key from the certificate chain using PubKeyFromCertChain(),
// and then uses the extracted public key to generate a peer ID using peer.IDFromPubKey().
// It returns the generated peer ID.
func EasyToUsePIDLoader(certs []*x509.Certificate) (peer.ID, error) {
	pk, err := PubKeyFromCertChain(certs)
	if err != nil {
		return "", err
	}
	return peer.IDFromPubKey(pk)
}
