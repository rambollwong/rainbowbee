package components

import (
	"errors"
	"sync"

	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowlog"
)

var (
	ErrSendStreamPoolHasSet = errors.New("send stream pool of the connection has been set")
	ErrSendStreamPoolNotSet = errors.New("send stream pool of the connection has not been set yet")

	_ manager.SendStreamPoolManager = (*SendStreamPoolManager)(nil)
)

// SendStreamPoolManager manages the send stream pools for peer connections.
type SendStreamPoolManager struct {
	mu      sync.RWMutex
	pools   map[peer.ID]map[network.Connection]manager.SendStreamPool
	connMgr manager.ConnectionManager
	logger  *rainbowlog.Logger
}

// Reset clears all send stream pools.
func (s *SendStreamPoolManager) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools = make(map[peer.ID]map[network.Connection]manager.SendStreamPool)
}

// AddPeerConnSendStreamPool adds a send stream pool for a specific peer connection.
func (s *SendStreamPoolManager) AddPeerConnSendStreamPool(pid peer.ID, conn network.Connection, sendStreamPool manager.SendStreamPool) error {
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
	return pool.Close()
}

// GetPeerBestConnSendStreamPool returns the send stream pool for the peer connection with the most idle streams.
func (s *SendStreamPoolManager) GetPeerBestConnSendStreamPool(pid peer.ID) manager.SendStreamPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	connM, ok := s.pools[pid]
	if !ok {
		return nil
	}
	var idleSize int
	var sendStreamPool manager.SendStreamPool
	for _, pool := range connM {
		idle := pool.IdleSize()
		if idleSize <= idle {
			sendStreamPool = pool
			idleSize = idle
		}
	}
	// todo need to add Connection expansion logic here?
	return sendStreamPool
}
