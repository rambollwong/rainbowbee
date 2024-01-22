package components

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/rambollwong/rainbowbee/components/payload"
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowbee/util"
	catutil "github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"
	"google.golang.org/protobuf/proto"
)

const (
	ProtocolExchangerProtocolID protocol.ID = "/protocol-exchanger/v0.0.1"
)

var (
	ErrUnknownConnectionDirection  = errors.New("unknown connection direction")
	ErrPushProtocolTimeout         = errors.New("push protocol timeout")
	ErrExchangeProtocolTimeout     = errors.New("exchange protocol timeout")
	ErrProtocolOfExchangerMismatch = errors.New("exchanger protocol mismatch")
	ErrPayloadTypeMismatch         = errors.New("exchanger payload type mismatch")
	exchangeTimeout                = 10 * time.Second
	pushRetryInterval              = time.Second

	_ manager.ProtocolExchanger = (*ProtocolExchanger)(nil)
)

type ProtocolExchanger struct {
	host        host.Host               // The host used for network communication.
	protocolMgr manager.ProtocolManager // The protocol manager for managing supported protocols.

	mu          sync.RWMutex              // Mutex for concurrent access to the pushSignalC map.
	pushSignalC map[peer.ID]chan struct{} // Map of peer IDs to push signal channels.

	logger *rainbowlog.Logger // Logger for logging events and errors.
}

func NewProtocolExchanger(h host.Host, protocolManager manager.ProtocolManager) manager.ProtocolExchanger {
	return &ProtocolExchanger{
		host:        h,
		protocolMgr: protocolManager,
		mu:          sync.RWMutex{},
		pushSignalC: make(map[peer.ID]chan struct{}),
		logger: log.Logger.SubLogger(
			rainbowlog.WithLabels(log.DefaultLoggerLabel, "PROTOCOL_EXCHANGER"),
		),
	}
}

// ProtocolID returns the protocol ID of the ProtocolExchanger.
func (p *ProtocolExchanger) ProtocolID() protocol.ID {
	return ProtocolExchangerProtocolID
}

// Handle returns the message payload handler for the ProtocolExchanger.
func (p *ProtocolExchanger) Handle() handler.MsgPayloadHandler {
	return func(senderPID peer.ID, msgPayload []byte) {
		msg := &payload.ProtocolExchangerPayload{}

		// Unmarshal the received message payload into the ProtocolExchangerPayload struct.
		if err := proto.Unmarshal(msgPayload, msg); err != nil {
			p.logger.Error().
				Msg("failed to unmarshal protocol exchanger payload.").
				Str("sender", senderPID.String()).
				Err(err).
				Done()
			return
		}

		// Check if the sender's peer ID matches the one specified in the payload.
		if senderPID.String() != msg.Pid {
			p.logger.Warn().
				Msg("sender pid mismatch.").
				Str("sender", senderPID.String()).
				Str("payload pid", msg.Pid).
				Done()
			return
		}

		p.logger.Info().
			Msg("protocol exchanger payload received.").
			Str("type", msg.PayloadType.String()).
			Str("sender", msg.Pid).
			Strs("protocols", msg.Protocols...).
			Done()

		// Convert the string array of protocols to protocol.ID array.
		protocols := util.ConvertStrArrToProtocolIDs(msg.Protocols...)

		// Handle the payload based on its type.
		switch msg.PayloadType {
		case payload.ProtocolExchangerPayload_PUSH:
			// Receive PUSH

			// Set the supported protocols of the sender peer.
			if err := p.protocolMgr.SetSupportedProtocolsOfPeer(senderPID, protocols); err != nil {
				p.logger.Error().
					Msg("failed to set supported protocols of peer.").
					Str("sender", senderPID.String()).
					Err(err).
					Done()
				return
			}

			// Send PUSH_OK response to the sender.
			if err := p.sendProtocolsToOther(senderPID, payload.ProtocolExchangerPayload_PUSH_OK); err != nil {
				p.logger.Error().
					Msg("failed to send PUSH_OK to sender.").
					Str("sender", senderPID.String()).
					Err(err).
					Done()
				return
			}

		case payload.ProtocolExchangerPayload_PUSH_OK:
			// Receive PUSH_OK

			// Retrieve the push signal channel associated with the sender peer.
			p.mu.RLock()
			pushSignalC, ok := p.pushSignalC[senderPID]
			p.mu.RUnlock()

			// Signal the push signal channel if it exists.
			if ok {
				pushSignalC <- struct{}{}
			}

			// Set the supported protocols of the sender peer.
			if err := p.protocolMgr.SetSupportedProtocolsOfPeer(senderPID, protocols); err != nil {
				p.logger.Error().
					Msg("failed to set supported protocols of peer.").
					Str("sender", senderPID.String()).
					Err(err).
					Done()
				return
			}
		}
	}
}

// ExchangeProtocol exchanges protocols with a connected network connection.
// It creates a timeout context and starts a goroutine to perform the protocol exchange.
// It waits for the exchange to complete or for the timeout to occur.
// Returns the exchanged protocols or an error if the operation times out.
func (p *ProtocolExchanger) ExchangeProtocol(conn network.Connection) (protocols []protocol.ID, err error) {
	ctx, cancelFunc := context.WithTimeout(p.host.Context(), exchangeTimeout)
	defer cancelFunc()
	signalC := make(chan struct{})

	go func() {
		protocols, err = p.exchangeProtocol(conn)

		// Notify the caller that the exchange is complete.
		select {
		case <-ctx.Done():
		case signalC <- struct{}{}:
		}
	}()

	// Wait for the exchange to complete or timeout.
	select {
	case <-ctx.Done():
		return nil, ErrExchangeProtocolTimeout
	case <-signalC:
		return protocols, err
	}
}

// PushProtocols sends a PUSH request to another peer identified by the given peer ID.
// It manages the synchronization of concurrent PushProtocols calls by using a pushSignalC map.
// If a PushProtocols call is already in progress for the same peer ID, it delays the execution for a certain period.
// Returns an error if the PUSH operation times out.
func (p *ProtocolExchanger) PushProtocols(pid peer.ID) error {
	p.mu.RLock()
	signalC, ok := p.pushSignalC[pid]
	p.mu.RUnlock()

	if ok {
		// Push is currently being executed.
		// Delay execution and retry after a certain interval.
		time.Sleep(pushRetryInterval)
		return p.PushProtocols(pid)
	} else {
		signalC = make(chan struct{}, 1)
		p.mu.Lock()
		p.pushSignalC[pid] = signalC
		p.mu.Unlock()
	}

	defer func() {
		// Cleanup the pushSignalC map after the PushProtocols operation is completed.
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.pushSignalC, pid)
	}()

	// Send PUSH request to the other peer.
	if err := p.sendProtocolsToOther(pid, payload.ProtocolExchangerPayload_PUSH); err != nil {
		p.logger.Debug().Msg("failed to send PUSH to other.").Str("remote", pid.String()).Err(err).Done()
		return err
	}

	// Wait for the PUSH operation to complete or timeout.
	select {
	case <-signalC:
		p.logger.Info().Msg("push protocols success.").Str("remote", pid.String()).Done()
		return nil
	case <-time.After(exchangeTimeout):
		return ErrPushProtocolTimeout
	}
}

// buildProtocolsPayload creates the payload for the protocol exchange message.
// It includes the local peer ID, registered protocols, and the payload type.
// Returns the marshaled payload or an error if marshaling fails.
func (p *ProtocolExchanger) buildProtocolsPayload(
	payloadType payload.ProtocolExchangerPayload_ProtocolExchangerPayloadType,
) ([]byte, error) {
	localPID := p.host.ID()
	protocols := p.protocolMgr.RegisteredAll()
	protocolsStrArr := util.ConvertProtocolIDsToStrArr(protocols...)
	sort.Strings(protocolsStrArr)
	msg := &payload.ProtocolExchangerPayload{
		Pid:         localPID.String(),
		Protocols:   protocolsStrArr,
		PayloadType: payloadType,
	}
	return proto.Marshal(msg)
}

// sendProtocolsToOther sends the protocols payload to another peer identified by the given peer ID.
// It builds the payload using buildProtocolsPayload and sends it using the host's SendMsg function.
// Returns an error if building the payload or sending the message fails.
func (p *ProtocolExchanger) sendProtocolsToOther(
	otherPID peer.ID,
	payloadType payload.ProtocolExchangerPayload_ProtocolExchangerPayloadType,
) error {
	payloadBz, err := p.buildProtocolsPayload(payloadType)
	if err != nil {
		return err
	}
	return p.host.SendMsg(ProtocolExchangerProtocolID, otherPID, payloadBz)
}

// sendExchangePayload sends an exchange payload over the provided sendStream.
// It builds the payload using the specified payloadType and writes it to the sendStream.
// Returns an error if there was a failure in building, marshaling, or writing the payload.
func (p *ProtocolExchanger) sendExchangePayload(
	sendStream network.SendStream,
	payloadType payload.ProtocolExchangerPayload_ProtocolExchangerPayloadType,
) (err error) {
	// Build the protocols payload
	payloadBz, err := p.buildProtocolsPayload(payloadType)
	if err != nil {
		p.logger.Debug().Msg("failed to build protocols payload.").Err(err).Done()
		return err
	}

	// Create a new payload package
	payloadPkg := protocol.NewPayloadPackage(ProtocolExchangerProtocolID, payloadBz, protocol.CompressTypeNone)

	// Marshal the payload package
	payloadPkgBz, err := payloadPkg.Marshal()
	if err != nil {
		p.logger.Debug().Msg("failed to marshal payload package.").Err(err).Done()
		return err
	}

	// Get the length of the payload package
	payloadPkgBzLengthBz := catutil.IntToBytes(len(payloadPkgBz))

	// Create a buffer to hold the payload data
	var buffer bytes.Buffer
	buffer.Write(payloadPkgBzLengthBz)
	buffer.Write(payloadPkgBz)

	// Write the payload to the sendStream
	if _, err = sendStream.Write(buffer.Bytes()); err != nil {
		p.logger.Debug().Msg("failed to write payload in send stream.").Err(err).Done()
		return err
	}

	return nil
}

// receiveExchangePayload receives an exchange payload from the provided receiveStream.
// It reads the payload from the receiveStream, un-marshals it, and validates the payload type.
// Returns the list of protocols extracted from the payload and an error if any.
func (p *ProtocolExchanger) receiveExchangePayload(
	receiveStream network.ReceiveStream,
	payloadType payload.ProtocolExchangerPayload_ProtocolExchangerPayloadType,
) (protocols []protocol.ID, err error) {
	// Read the length of the payload package from the receiveStream
	dataLength, _, err := util.ReadPackageLength(receiveStream)
	if err != nil {
		p.logger.Debug().Msg("failed to read package length.").
			Str("dir", receiveStream.Conn().Direction().String()).
			Str("remote", receiveStream.Conn().RemotePeerID().String()).
			Err(err).
			Done()
		return nil, err
	}

	// Read the payload data from the receiveStream
	dataBz, err := util.ReadPackageData(receiveStream, dataLength)
	if err != nil {
		p.logger.Debug().Msg("failed to read package data.").
			Str("dir", receiveStream.Conn().Direction().String()).
			Str("remote", receiveStream.Conn().RemotePeerID().String()).
			Err(err).
			Done()
		return nil, err
	}

	// Unmarshal the payload package
	payloadPkg := &protocol.PayloadPkg{}
	if err = payloadPkg.Unmarshal(dataBz); err != nil {
		p.logger.Debug().Msg("failed to unmarshal payload package.").
			Str("dir", receiveStream.Conn().Direction().String()).
			Str("remote", receiveStream.Conn().RemotePeerID().String()).
			Err(err).
			Done()
		return nil, err
	}

	// Check if the protocol ID of the payload package matches the exchanger's protocol ID
	if payloadPkg.ProtocolID() != p.ProtocolID() {
		return nil, ErrProtocolOfExchangerMismatch
	}

	// Unmarshal the payload into a ProtocolExchangerPayload
	exchangerPayload := &payload.ProtocolExchangerPayload{}
	if err = proto.Unmarshal(payloadPkg.Payload(), exchangerPayload); err != nil {
		p.logger.Debug().Msg("failed to unmarshal protocol exchanger payload package.").
			Str("dir", receiveStream.Conn().Direction().String()).
			Str("remote", receiveStream.Conn().RemotePeerID().String()).
			Err(err).
			Done()
		return nil, err
	}

	// Check if the payload type matches the expected payloadType
	if exchangerPayload.GetPayloadType() != payloadType {
		return nil, ErrPayloadTypeMismatch
	}

	// Transform the list of protocols into protocol.ID type
	return catutil.SliceTransformType(exchangerPayload.GetProtocols(), func(_ int, item string) protocol.ID {
		return protocol.ID(item)
	}), nil
}

// exchangeProtocol performs the protocol exchange on the provided connection.
// It sends a REQUEST payload in the case of an outbound connection and expects a RESPONSE payload.
// For an inbound connection, it receives a REQUEST payload and sends a RESPONSE payload.
// Returns the exchanged protocols and any error encountered during the process.
func (p *ProtocolExchanger) exchangeProtocol(conn network.Connection) ([]protocol.ID, error) {
	switch conn.Direction() {
	case network.Inbound:
		// Inbound connection: receive REQUEST payload
		receiveStream, err := conn.AcceptReceiveStream()
		if err != nil {
			p.logger.Debug().Msg("failed to accept a receive stream.").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}
		defer receiveStream.Close()

		// Receive and process the REQUEST payload
		protocols, err := p.receiveExchangePayload(receiveStream, payload.ProtocolExchangerPayload_REQUEST)
		if err != nil {
			p.logger.Debug().Msg("failed to receive exchange payload[REQUEST].").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}

		// send RESPONSE payload
		sendStream, err := conn.OpenSendStream()
		if err != nil {
			p.logger.Debug().Msg("failed to open a send stream.").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}
		defer sendStream.Close()

		// Send the RESPONSE payload
		if err = p.sendExchangePayload(sendStream, payload.ProtocolExchangerPayload_RESPONSE); err != nil {
			p.logger.Debug().Msg("failed to send exchange payload[RESPONSE].").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}

		return protocols, nil

	case network.Outbound:
		// Outbound connection: send REQUEST payload
		sendStream, err := conn.OpenSendStream()
		if err != nil {
			p.logger.Debug().Msg("failed to open a send stream.").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}
		defer sendStream.Close()

		// Send the REQUEST payload
		if err = p.sendExchangePayload(sendStream, payload.ProtocolExchangerPayload_REQUEST); err != nil {
			p.logger.Debug().Msg("failed to send exchange payload[REQUEST].").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}

		// Receive RESPONSE payload
		receiveStream, err := conn.AcceptReceiveStream()
		if err != nil {
			p.logger.Debug().Msg("failed to accept a receive stream.").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}
		defer receiveStream.Close()

		// Receive and process the RESPONSE payload
		protocols, err := p.receiveExchangePayload(receiveStream, payload.ProtocolExchangerPayload_RESPONSE)
		if err != nil {
			p.logger.Debug().Msg("failed to receive exchange payload[RESPONSE].").
				Str("dir", conn.Direction().String()).
				Err(err).
				Done()
			return nil, err
		}

		return protocols, nil

	default:
		return nil, ErrUnknownConnectionDirection
	}
}
