package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
)

func TestECDSABasicSignAndVerify(t *testing.T) {
	pri, pub, err := GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("hello! and welcome to some awesome crypto primitives")

	sig, err := pri.Sign(data)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := pub.Verify(data, sig)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("signature didn't match")
	}

	// change data
	data[0] = ^data[0]
	ok, err = pub.Verify(data, sig)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("signature matched and shouldn't")
	}
}

func TestECDSASignZero(t *testing.T) {
	pri, pub, err := GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 0)
	sig, err := pri.Sign(data)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := pub.Verify(data, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("signature didn't match")
	}
}

func TestECDSAMarshalLoop(t *testing.T) {
	pri, pub, err := GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	privB, err := ProtoMarshalPriKey(pri)
	if err != nil {
		t.Fatal(err)
	}

	privNew, err := ProtoUnmarshalPriKey(privB)
	if err != nil {
		t.Fatal(err)
	}

	if !pri.Equals(privNew) || !privNew.Equals(pri) {
		t.Fatal("keys are not equal")
	}

	pubB, err := ProtoMarshalPubKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	pubNew, err := ProtoUnmarshalPubKey(pubB)
	if err != nil {
		t.Fatal(err)
	}

	if !pub.Equals(pubNew) || !pubNew.Equals(pub) {
		t.Fatal("keys are not equal")
	}

}

func TestECDSAPublicKeyFromPubKey(t *testing.T) {
	ecdsaPrivK, err := ecdsa.GenerateKey(ECDSACurve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	privK, _, err := ECDSAKeyPairFromKey(ecdsaPrivK)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("Hello world!")
	signature, err := privK.Sign(data)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := ECDSAPublicKeyFromPubKey(ecdsaPrivK.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := pubKey.Verify(data, signature)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("signature didn't match")
	}

	pubB, err := ProtoMarshalPubKey(pubKey)
	if err != nil {
		t.Fatal(err)
	}
	pubNew, err := ProtoUnmarshalPubKey(pubB)
	if err != nil {
		t.Fatal(err)
	}

	if !pubKey.Equals(pubNew) || !pubNew.Equals(pubKey) {
		t.Fatal("keys are not equal")
	}
}
