package protocol

import (
	"bytes"
	"errors"

	"github.com/rambollwong/rainbowcat/util"
)

// CompressType represents the type of compression used for payload.
type CompressType uint8

const (
	CompressTypeNone CompressType = iota // No compression
	CompressTypeGzip                     // Gzip compression
)

var (
	// ErrUnknownCompressType is returned when encountering an unknown compress type.
	ErrUnknownCompressType = errors.New("unknown compress type")

	// ErrInvalidDataLength is returned when the data length is invalid.
	ErrInvalidDataLength = errors.New("invalid data length")
)

// PayloadPackage represents a protocol payload package.
type PayloadPackage interface {
	ProtocolID() ID           // Returns the protocol ID.
	Payload() []byte          // Returns the payload data.
	Marshal() ([]byte, error) // Marshals the payload package into a byte slice.
	Unmarshal([]byte) error   // Unmarshals the byte slice into the payload package.
}

// PayloadPkg represents a protocol payload package.
type PayloadPkg struct {
	protocol     ID           // Protocol ID
	payload      []byte       // Payload data
	compressType CompressType // Compression type
}

func NewPayloadPackage(protocol ID, payload []byte, compressType CompressType) PayloadPackage {
	return &PayloadPkg{
		protocol:     protocol,
		payload:      payload,
		compressType: compressType,
	}
}

// ProtocolID returns the protocol ID of the payload package.
func (p *PayloadPkg) ProtocolID() ID {
	return p.protocol
}

// Payload returns the payload data of the payload package.
func (p *PayloadPkg) Payload() []byte {
	return p.payload
}

// Marshal marshals the payload package into a byte slice.
func (p *PayloadPkg) Marshal() (bz []byte, err error) {
	protocolLength := len(p.protocol)
	var finalPayload []byte
	switch p.compressType {
	case CompressTypeNone:
		finalPayload = p.payload
	case CompressTypeGzip:
		finalPayload, err = util.GZipCompressBytes(p.payload)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnknownCompressType
	}
	payloadLength := len(finalPayload)
	bz = make([]byte, 0, 8+protocolLength+8+payloadLength+1)
	bz = append(bz, util.IntToBytes(protocolLength)...)
	bz = append(bz, p.protocol...)
	bz = append(bz, util.IntToBytes(payloadLength)...)
	bz = append(bz, finalPayload...)
	bz = append(bz, byte(p.compressType))
	return bz, err
}

// Unmarshal unmarshals the byte slice into the payload package.
func (p *PayloadPkg) Unmarshal(bz []byte) (err error) {
	if len(bz) <= 17 {
		return ErrInvalidDataLength
	}
	reader := bytes.NewReader(bz)
	// Read protocol ID
	lengthBz := make([]byte, 8)
	_, err = reader.Read(lengthBz)
	if err != nil {
		return err
	}
	protocolLength := util.BytesToInt(lengthBz)
	protocolBz := make([]byte, protocolLength)
	_, err = reader.Read(protocolBz)
	if err != nil {
		return err
	}
	p.protocol = ID(protocolBz)
	// Read payload
	_, err = reader.Read(lengthBz)
	if err != nil {
		return err
	}
	payloadLength := util.BytesToInt(lengthBz)
	payloadBz := make([]byte, payloadLength)
	_, err = reader.Read(payloadBz)
	if err != nil {
		return err
	}
	// Read compress type
	var compressType byte
	compressType, err = reader.ReadByte()
	if err != nil {
		return err
	}
	p.compressType = CompressType(compressType)
	switch p.compressType {
	case CompressTypeNone:
		p.payload = payloadBz
	case CompressTypeGzip:
		p.payload, err = util.GZipDecompressBytes(payloadBz)
		if err != nil {
			return err
		}
	default:
		return ErrUnknownCompressType
	}
	return nil
}
