package components

import (
	"errors"
	"sync"

	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowcat/types"
	"github.com/rambollwong/rainbowlog"
)

var (
	ErrMaxReceiveStreamsCountReached = errors.New("max receive streams count reached")

	_ manager.ReceiveStreamManager = (*ReceiveStreamManager)(nil)
)

// ReceiveStreamManager manages the receive streams for peer connections.
type ReceiveStreamManager struct {
	mu        sync.RWMutex
	maxCount  int
	countMap  map[peer.ID]int
	streamMap map[peer.ID]map[network.Connection]*types.Set[network.ReceiveStream]
	logger    *rainbowlog.Logger
}

// NewReceiveStreamManager creates a new instance of the ReceiveStreamManager type
// that implements the manager.ReceiveStreamManager interface.
// The maxCountEachPeer parameter specifies the maximum count of receive streams allowed per peer.
// The returned ReceiveStreamManager can be used to manage receive streams for peer connections.
func NewReceiveStreamManager(maxCountEachPeer int) manager.ReceiveStreamManager {
	return &ReceiveStreamManager{
		mu:        sync.RWMutex{},
		maxCount:  maxCountEachPeer,
		countMap:  make(map[peer.ID]int),
		streamMap: make(map[peer.ID]map[network.Connection]*types.Set[network.ReceiveStream]),
		logger: log.Logger.SubLogger(
			rainbowlog.WithLabels(log.DefaultLoggerLabel, "RECEIVE-STREAM-MANAGER"),
		),
	}
}

// Reset clears all receive streams and counts.
func (r *ReceiveStreamManager) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.countMap = make(map[peer.ID]int)
	r.streamMap = make(map[peer.ID]map[network.Connection]*types.Set[network.ReceiveStream])
	r.logger.Debug().Msg("reset.").Done()
}

// SetPeerReceiveStreamMaxCount sets the maximum count of receive streams for a peer.
func (r *ReceiveStreamManager) SetPeerReceiveStreamMaxCount(max int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maxCount = max
	r.logger.Debug().Msg("max count set").Int("set", max).Done()
}

// AddPeerReceiveStream adds a receive stream for a specific peer connection.
func (r *ReceiveStreamManager) AddPeerReceiveStream(
	pid peer.ID,
	conn network.Connection,
	receiveStream network.ReceiveStream,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	count, _ := r.countMap[pid]
	if count >= r.maxCount {
		return ErrMaxReceiveStreamsCountReached
	}

	connM, ok := r.streamMap[pid]
	if !ok {
		connM = make(map[network.Connection]*types.Set[network.ReceiveStream])
		r.streamMap[pid] = connM
	}
	streamSet, ok := connM[conn]
	if !ok {
		streamSet = types.NewSet[network.ReceiveStream]()
		connM[conn] = streamSet
	}
	ok = streamSet.Put(receiveStream)
	if ok {
		count++
		r.countMap[pid] = count
	}
	r.logger.Debug().Msg("receive stream added").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Done()
	return nil
}

// RemovePeerReceiveStream removes a receive stream for a specific peer connection.
func (r *ReceiveStreamManager) RemovePeerReceiveStream(
	pid peer.ID,
	conn network.Connection,
	receiveStream network.ReceiveStream,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	count, _ := r.countMap[pid]
	if count == 0 {
		return nil
	}

	connM, ok := r.streamMap[pid]
	if !ok {
		return nil
	}
	streamSet, ok := connM[conn]
	if !ok {
		return nil
	}
	ok = streamSet.Remove(receiveStream)
	if ok {
		count--
		if count == 0 {
			delete(r.countMap, pid)
			delete(r.streamMap, pid)
		} else {
			r.countMap[pid] = count
		}
	}
	r.logger.Debug().Msg("receive stream removed").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Done()
	return nil
}

// GetCurrentPeerReceiveStreamCount returns the current count of receive streams for a peer.
func (r *ReceiveStreamManager) GetCurrentPeerReceiveStreamCount(pid peer.ID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.countMap[pid]
}

// ClosePeerReceiveStreams closes all receive streams for a specific peer connection.
func (r *ReceiveStreamManager) ClosePeerReceiveStreams(pid peer.ID, conn network.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	count, _ := r.countMap[pid]
	if count == 0 {
		return nil
	}

	connM, ok := r.streamMap[pid]
	if !ok {
		return nil
	}
	streamSet, ok := connM[conn]
	if !ok {
		return nil
	}
	streamSet.Range(func(s network.ReceiveStream) bool {
		count--
		_ = s.Close()
		return true
	})
	if count == 0 {
		delete(r.countMap, pid)
		delete(r.streamMap, pid)
	} else {
		r.countMap[pid] = count
		delete(connM, conn)
	}
	r.logger.Debug().Msg("all receive streams are closed.").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Done()
	return nil
}
