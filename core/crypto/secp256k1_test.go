package crypto

import (
	"crypto/rand"
	"testing"
)

func TestSecp256k1BasicSignAndVerify(t *testing.T) {
	pri, pub, err := GenerateSecp256k1Key(rand.Reader)
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

func TestSecp256k1SignZero(t *testing.T) {
	pri, pub, err := GenerateSecp256k1Key(rand.Reader)
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

func TestSecp256k1MarshalLoop(t *testing.T) {
	pri, pub, err := GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	priB, err := ProtoMarshalPriKey(pri)
	if err != nil {
		t.Fatal(err)
	}

	priNew, err := ProtoUnmarshalPriKey(priB)
	if err != nil {
		t.Fatal(err)
	}

	if !pri.Equals(priNew) || !priNew.Equals(pri) {
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
