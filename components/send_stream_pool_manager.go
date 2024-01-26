package components

import (
	"errors"
	"sync"

	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowlog"
)

var (
	ErrSendStreamPoolHasSet = errors.New("send stream pool of the connection has been set")
	ErrSendStreamPoolNotSet = errors.New("send stream pool of the connection has not been set yet")

	_ manager.SendStreamPoolManager = (*SendStreamPoolManager)(nil)
)

// SendStreamPoolManager manages the send stream pools for peer connections.
type SendStreamPoolManager struct {
	mu     sync.RWMutex
	pools  map[peer.ID]map[network.Connection]manager.SendStreamPool
	logger *rainbowlog.Logger
}

// NewSendStreamPoolManager creates a new instance of the SendStreamPoolManager type
// that implements the manager.SendStreamPoolManager interface.
// The returned SendStreamPoolManager can be used to manage send stream pools.
func NewSendStreamPoolManager() *SendStreamPoolManager {
	return &SendStreamPoolManager{
		mu:    sync.RWMutex{},
		pools: make(map[peer.ID]map[network.Connection]manager.SendStreamPool),
		logger: log.Logger.SubLogger(
			rainbowlog.WithLabels(log.DefaultLoggerLabel, "SEND-STREAM-POOL-MANAGER"),
		),
	}
}

func (s *SendStreamPoolManager) AttachHost(_ host.Host) {

}

// Reset clears all send stream pools.
func (s *SendStreamPoolManager) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools = make(map[peer.ID]map[network.Connection]manager.SendStreamPool)
	s.logger.Debug().Msg("reset.").Done()
}

// AddPeerConnSendStreamPool adds a send stream pool for a specific peer connection.
func (s *SendStreamPoolManager) AddPeerConnSendStreamPool(
	pid peer.ID,
	conn network.Connection,
	sendStreamPool manager.SendStreamPool,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	connM, ok := s.pools[pid]
	if !ok {
		connM = make(map[network.Connection]manager.SendStreamPool)
		s.pools[pid] = connM
	}
	_, ok = connM[conn]
	if ok {
		return ErrSendStreamPoolHasSet
	}
	connM[conn] = sendStreamPool
	s.logger.Debug().Msg("send stream pool added.").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Done()
	return nil
}

// RemovePeerConnAndCloseSendStreamPool removes a send stream pool for a specific peer connection and closes the pool.
func (s *SendStreamPoolManager) RemovePeerConnAndCloseSendStreamPool(pid peer.ID, conn network.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	connM, ok := s.pools[pid]
	if !ok {
		return ErrSendStreamPoolNotSet
	}
	pool, ok := connM[conn]
	if !ok {
		return ErrSendStreamPoolNotSet
	}
	delete(connM, conn)
	if len(connM) == 0 {
		delete(s.pools, pid)
	}
	err := pool.Close()
	s.logger.Debug().Msg("conn removed and send stream pool closed.").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Err(err).
		Done()
	return err
}

// GetPeerBestConnSendStreamPool returns the send stream pool for the peer connection with the most idle streams.
func (s *SendStreamPoolManager) GetPeerBestConnSendStreamPool(pid peer.ID) manager.SendStreamPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	connM, ok := s.pools[pid]
	if !ok {
		return nil
	}
	var (
		idleSize       int
		sendStreamPool manager.SendStreamPool
		conn           network.Connection
	)

	for c, pool := range connM {
		idle := pool.IdleSize()
		if idleSize <= idle {
			sendStreamPool = pool
			idleSize = idle
			conn = c
		}
	}
	// todo need to add Connection expansion logic here?
	s.logger.Debug().Msg("best send stream pool of connections got.").
		Str("pid", pid.String()).
		Str("conn-remote-addr", conn.RemoteAddr().String()).
		Int("idle-size", idleSize).
		Done()
	return sendStreamPool
}
