package rainbowbee

import (
	"sync/atomic"

	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowcat/pipeline"
)

var (
	taskIndex uint64 = 0 // taskIndex is used to mark tasks in the pipeline.
)

// pkgBzRemotePIDPair represents a pair of a byte slice and a peer ID.
type pkgBzRemotePIDPair struct {
	taskIndex uint64
	pkgBz     []byte
	remotePID peer.ID
}

// pkgRemotePIDPair represents a pair of a protocol payload package and a peer ID.
type pkgRemotePIDPair struct {
	taskIndex uint64
	pkg       protocol.PayloadPackage
	remotePID peer.ID
}

// payloadToBeHandled represents a payload to be handled, including a message payload handler,
// the payload itself, and the remote peer ID.
type payloadToBeHandled struct {
	taskIndex  uint64
	handler    handler.MsgPayloadHandler
	payload    []byte
	remotePID  peer.ID
	protocolID protocol.ID
}

// Call invokes the message payload handler with the remote peer ID and payload.
func (p payloadToBeHandled) Call() {
	p.handler(p.remotePID, p.payload)
}

// nextTaskIndex returns the next taskIndex.
func nextTaskIndex() uint64 {
	return atomic.AddUint64(&taskIndex, 1)
}

// handleReceiveStreamDataTask is a task provider for the handleMsgPayloadPipeline.
// It takes an input of type *pkgBzRemotePIDPair and returns an output of type *pkgRemotePIDPair.
func (h *Host) handleReceiveStreamDataTask() pipeline.GenericTaskProvider[*pkgBzRemotePIDPair, *pkgRemotePIDPair] {
	return func(input *pkgBzRemotePIDPair) (output *pkgRemotePIDPair, ok bool) {
		input.taskIndex = nextTaskIndex()
		h.logger.Debug().
			Msg("handling receive stream data.").
			Uint64("index", input.taskIndex).
			Str("sender-pid", input.remotePID.String()).
			Int("payload-length", len(input.pkgBz)).
			Done()
		// If the received data is empty, return without further processing.
		if input == nil || len(input.pkgBz) == 0 {
			return nil, false
		}

		// Create an empty PayloadPkg struct to unmarshal the received data.
		pkg := &protocol.PayloadPkg{}

		// Unmarshal the received data into the PayloadPkg struct.
		if err := pkg.Unmarshal(input.pkgBz); err != nil {
			// If unmarshalling fails, log a warning message and drop the payload.
			h.logger.Warn().
				Msg("failed to unmarshal payload package, drop it.").
				Str("remote_pid", input.remotePID.String()).
				Err(err).
				Done()
			return nil, false
		}

		return &pkgRemotePIDPair{
			taskIndex: input.taskIndex,
			pkg:       pkg,
			remotePID: input.remotePID,
		}, true
	}
}

// routePayloadToHandlerTask is a task provider for the handleMsgPayloadPipeline.
// It takes an input of type *pkgRemotePIDPair and returns an output of type *payloadToBeHandled.
func (h *Host) routePayloadToHandlerTask() pipeline.GenericTaskProvider[*pkgRemotePIDPair, *payloadToBeHandled] {
	return func(input *pkgRemotePIDPair) (output *payloadToBeHandled, ok bool) {
		h.logger.Debug().
			Msg("routing payload to handler.").
			Uint64("index", input.taskIndex).
			Str("sender-pid", input.remotePID.String()).
			Str("protocol", input.pkg.ProtocolID().String()).
			Done()
		// Retrieve the payload handler associated with the protocol ID of the payload package.
		payloadHandler := h.protocolMgr.Handler(input.pkg.ProtocolID())

		// If the payload handler is not found (nil), log a warning message and drop the payload.
		if payloadHandler == nil {
			h.logger.Warn().
				Msg("payload handler not found, drop the payload.").
				Str("protocol", input.pkg.ProtocolID().String()).
				Str("remote_pid", input.remotePID.String()).
				Done()
			return nil, false
		}

		return &payloadToBeHandled{
			taskIndex:  input.taskIndex,
			handler:    payloadHandler,
			payload:    input.pkg.Payload(),
			remotePID:  input.remotePID,
			protocolID: input.pkg.ProtocolID(),
		}, true
	}
}

// callHandlerTask is a task provider for the handleMsgPayloadPipeline.
// It takes an input of type *payloadToBeHandled and returns an output of type struct{}.
func (h *Host) callHandlerTask() pipeline.GenericTaskProvider[*payloadToBeHandled, struct{}] {
	return func(input *payloadToBeHandled) (output struct{}, ok bool) {
		h.logger.Debug().
			Msg("calling handler.").
			Uint64("index", input.taskIndex).
			Str("sender-pid", input.remotePID.String()).
			Done()
		input.Call()
		return struct{}{}, true
	}
}

// runHandleMsgPipeline sets up and runs the message payload handling pipeline.
// It uses the RunParallelTaskPipeline function to create a parallel task pipeline
// with the specified concurrency and task providers.
// The pipeline consists of the handleReceiveStreamDataTask, routePayloadToHandlerTask, and callHandlerTask tasks.
func (h *Host) runHandleMsgPipeline() error {
	handleMsgPayloadPipeline, err := pipeline.RunParallelTaskPipeline(
		pipelineCount,
		[]uint8{
			h.cfg.PayloadUnmarshalerConcurrency,
			h.cfg.PayloadHandlerRouterConcurrency,
			h.cfg.HandlerExecutorConcurrency,
		},
		h.handleReceiveStreamDataTask(),
		h.routePayloadToHandlerTask(),
		h.callHandlerTask(),
	)
	if err != nil {
		return err
	}
	h.handleMsgPayloadPipeline = handleMsgPayloadPipeline.NoOutput()
	return nil
}
