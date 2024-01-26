package components

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"

	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowcat/types"
	"github.com/rambollwong/rainbowlog"
)

var (
	_ manager.ConnectionManager = (*LevelConnectionManager)(nil)
)

const (
	EliminationStrategyUnknown EliminationStrategy = iota
	EliminationStrategyRandom
	EliminationStrategyFIFO
	EliminationStrategyLIFO

	DefaultMaxCountOfPeers               = 20
	DefaultMaxCountOfConnectionsEachPeer = 1
	DefaultEliminationStrategy           = EliminationStrategyLIFO
)

// EliminationStrategy represents the type of elimination strategy.
type EliminationStrategy uint8

// String returns the string representation of the elimination strategy.
func (e EliminationStrategy) String() string {
	switch e {
	case EliminationStrategyRandom:
		return "RANDOM"
	case EliminationStrategyFIFO:
		return "FIFO"
	case EliminationStrategyLIFO:
		return "LIFO"
	default:
		return "UNKNOWN"
	}
}

// StrToEliminationStrategy converts a string to the corresponding EliminationStrategy.
func StrToEliminationStrategy(str string) EliminationStrategy {
	switch strings.ToLower(str) {
	case "random":
		return EliminationStrategyRandom
	case "fifo":
		return EliminationStrategyFIFO
	case "lifo":
		return EliminationStrategyLIFO
	default:
		return EliminationStrategyUnknown
	}
}

type peerConnections struct {
	pid     peer.ID
	connSet *types.Set[network.Connection]
}

type LevelConnectionManager struct {
	mu                            sync.RWMutex
	maxCountOfPeers               int
	maxCountOfConnectionsEachPeer int64
	strategy                      EliminationStrategy
	highLevelPeersLock            sync.RWMutex
	highLevelPeers                *types.Set[peer.ID]
	highLevelConn                 []*peerConnections
	lowLevelConn                  []*peerConnections

	h          host.Host
	expandingC chan struct{}

	eliminatePeers *types.Set[peer.ID]

	logger *rainbowlog.Logger
}

func NewLevelConnectionManager() *LevelConnectionManager {
	return &LevelConnectionManager{
		mu:                            sync.RWMutex{},
		maxCountOfPeers:               DefaultMaxCountOfPeers,
		maxCountOfConnectionsEachPeer: DefaultMaxCountOfConnectionsEachPeer,
		strategy:                      DefaultEliminationStrategy,
		highLevelPeersLock:            sync.RWMutex{},
		highLevelPeers:                types.NewSet[peer.ID](),
		highLevelConn:                 make([]*peerConnections, 0, 10),
		lowLevelConn:                  make([]*peerConnections, 0, 10),
		h:                             nil,
		expandingC:                    make(chan struct{}, 1),
		eliminatePeers:                types.NewSet[peer.ID](),
		logger: log.Logger.SubLogger(
			rainbowlog.WithLabels(log.DefaultLoggerLabel, "LEVEL-CONNECTION-MANAGER"),
		),
	}
}

func (l *LevelConnectionManager) AttachHost(h host.Host) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.h = h
}

// SetStrategy sets the elimination strategy for the LevelConnectionManager.
func (l *LevelConnectionManager) SetStrategy(strategy EliminationStrategy) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch strategy {
	case EliminationStrategyRandom, EliminationStrategyFIFO, EliminationStrategyLIFO:
		l.strategy = strategy
	default:
		l.logger.Warn().
			Msg("wrong elimination strategy set, use default strategy.").
			Str("got", strategy.String()).
			Str("use", DefaultEliminationStrategy.String()).
			Done()
		l.strategy = DefaultEliminationStrategy
	}
}

// SetMaxCountOfPeers sets the maximum count of peers for the LevelConnectionManager.
func (l *LevelConnectionManager) SetMaxCountOfPeers(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if max < 1 {
		l.logger.Warn().
			Msg("wrong max count of peers set, use default max count.").
			Int("got", max).
			Int("use", DefaultMaxCountOfPeers).
			Done()
		max = DefaultMaxCountOfPeers
	}
	l.maxCountOfPeers = max
}

// SetMaxCountOfConnectionsEachPeer sets the maximum count of connections allowed for each peer in the LevelConnectionManager.
func (l *LevelConnectionManager) SetMaxCountOfConnectionsEachPeer(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if max < 1 {
		l.logger.Warn().
			Msg("wrong max count of connections each peer, use default max count").
			Int("got", max).
			Int("use", DefaultMaxCountOfConnectionsEachPeer).
			Done()
		max = DefaultMaxCountOfConnectionsEachPeer
	}
	l.maxCountOfConnectionsEachPeer = int64(max)
}

// AddHighLevelPeer adds a high-level peer ID to the LevelConnectionManager.
func (l *LevelConnectionManager) AddHighLevelPeer(pid peer.ID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.highLevelPeers.Put(pid)
}

// getHighLevelConnections returns the connection set and index of a peer in the high level connections.
func (l *LevelConnectionManager) getHighLevelConnections(pid peer.ID) (*types.Set[network.Connection], int) {
	for idx, connections := range l.highLevelConn {
		if pid == connections.pid {
			return connections.connSet, idx
		}
	}
	return nil, -1
}

// getLowLevelConnections returns the connection set and index of a peer in the low level connections.
func (l *LevelConnectionManager) getLowLevelConnections(pid peer.ID) (*types.Set[network.Connection], int) {
	for idx, connections := range l.lowLevelConn {
		if pid == connections.pid {
			return connections.connSet, idx
		}
	}
	return nil, -1
}

// closeHighLevelConnWithIdx closes all connections in the high level connections set at the given index.
func (l *LevelConnectionManager) closeLowLevelConnWithIdx(idx int) {
	l.lowLevelConn[idx].connSet.Range(func(c network.Connection) bool {
		go func(connToClose network.Connection) {
			_ = connToClose.Close()
		}(c)
		return true
	})
}

// closeLowLevelConnRandom closes a random connection in the low level connections set and returns the eliminated peer ID.
func (l *LevelConnectionManager) closeHighLevelConnWithIdx(idx int) {
	l.highLevelConn[idx].connSet.Range(func(c network.Connection) bool {
		go func(connToClose network.Connection) {
			_ = connToClose.Close()
		}(c)
		return true
	})
}

// closeLowLevelConnRandom chooses a random low-level connection, closes it, and removes it from the connection manager.
func (l *LevelConnectionManager) closeLowLevelConnRandom(lowLevelConnCount int) peer.ID {
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(lowLevelConnCount)))
	random := int(r.Int64())
	eliminatedPID := l.lowLevelConn[random].pid
	l.closeLowLevelConnWithIdx(random)
	if random == lowLevelConnCount-1 {
		l.lowLevelConn = l.lowLevelConn[:random]
	} else {
		l.lowLevelConn = append(l.lowLevelConn[:random], l.lowLevelConn[random+1:]...)
	}
	return eliminatedPID
}

// closeHighLevelConnRandom chooses a random high-level connection, closes it, and removes it from the connection manager.
func (l *LevelConnectionManager) closeHighLevelConnRandom(highLevelConnCount int) peer.ID {
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(highLevelConnCount)))
	random := int(r.Int64())
	eliminatedPID := l.highLevelConn[random].pid
	l.closeHighLevelConnWithIdx(random)
	if random == highLevelConnCount-1 {
		l.highLevelConn = l.highLevelConn[:random]
	} else {
		l.highLevelConn = append(l.highLevelConn[:random], l.highLevelConn[random+1:]...)
	}
	return eliminatedPID
}

// eliminateConnectionsRandom eliminates a random connection based on the provided strategy (low or high level).
// It returns the eliminated peer's ID and a boolean indicating if the elimination was successful.
func (l *LevelConnectionManager) eliminateConnectionsRandom(eliminateHigh bool) (peer.ID, bool) {
	hCount := len(l.highLevelConn)
	lCount := len(l.lowLevelConn)
	if hCount+lCount > l.maxCountOfPeers {
		if lCount > 0 {
			eliminatedPID := l.closeLowLevelConnRandom(lCount)
			l.logger.Info().
				Msg("connection eliminated").
				Str("strategy", "random").
				Str("level", "low").
				Str("pid", eliminatedPID.String()).
				Done()
			return eliminatedPID, true
		}
		if !eliminateHigh {
			return "", false
		}
		eliminatedPID := l.closeHighLevelConnRandom(hCount)
		l.logger.Info().
			Msg("connection eliminated").
			Str("strategy", "random").
			Str("level", "high").
			Str("pid", eliminatedPID.String()).
			Done()
		return eliminatedPID, true
	}
	return "", true
}

// closeLowLevelConnFirst closes the first low-level connection and returns the eliminated peer ID.
func (l *LevelConnectionManager) closeLowLevelConnFirst() peer.ID {
	eliminatedPID := l.lowLevelConn[0].pid
	l.closeLowLevelConnWithIdx(0)
	l.lowLevelConn = l.lowLevelConn[1:]
	l.logger.Info().
		Msg("connection eliminated").
		Str("strategy", "fifo").
		Str("level", "low").
		Str("pid", eliminatedPID.String()).
		Done()
	return eliminatedPID
}

// closeHighLevelConnFirst closes the first high-level connection and returns the eliminated peer ID.
func (l *LevelConnectionManager) closeHighLevelConnFirst() peer.ID {
	eliminatedPID := l.highLevelConn[0].pid
	l.closeHighLevelConnWithIdx(0)
	l.highLevelConn = l.highLevelConn[1:]
	l.logger.Info().
		Msg("connection eliminated").
		Str("strategy", "fifo").
		Str("level", "high").
		Str("pid", eliminatedPID.String()).
		Done()
	return eliminatedPID
}

// eliminateConnectionsFIFO eliminates connections using the FIFO (First-In-First-Out) strategy.
// If eliminateHigh is true, it prioritizes eliminating high-level connections.
// It returns the eliminated peer ID and a flag indicating successful elimination.
func (l *LevelConnectionManager) eliminateConnectionsFIFO(eliminateHigh bool) (peer.ID, bool) {
	hCount := len(l.highLevelConn)
	lCount := len(l.lowLevelConn)

	if hCount+lCount > l.maxCountOfPeers {
		if lCount > 0 {
			eliminatedPID := l.closeLowLevelConnFirst()
			return eliminatedPID, true
		}

		if !eliminateHigh {
			return "", false
		}

		eliminatedPID := l.closeHighLevelConnFirst()
		return eliminatedPID, true
	}

	return "", true
}

// closeLowLevelConnLast closes the last low-level connection and returns the eliminated peer ID.
func (l *LevelConnectionManager) closeLowLevelConnLast(lowLevelConnCount int) peer.ID {
	idx := lowLevelConnCount - 1
	eliminatedPID := l.lowLevelConn[idx].pid
	l.closeLowLevelConnWithIdx(idx)
	l.lowLevelConn = l.lowLevelConn[0:idx]
	l.logger.Info().
		Msg("connection eliminated").
		Str("strategy", "lifo").
		Str("level", "low").
		Str("pid", eliminatedPID.String()).
		Done()
	return eliminatedPID
}

// closeHighLevelConnLast closes the last high-level connection and returns the eliminated peer ID.
func (l *LevelConnectionManager) closeHighLevelConnLast(highLevelConnCount int) peer.ID {
	idx := highLevelConnCount - 1
	eliminatedPID := l.highLevelConn[idx].pid
	l.closeHighLevelConnWithIdx(idx)
	l.highLevelConn = l.highLevelConn[0:idx]
	l.logger.Info().
		Msg("connection eliminated").
		Str("strategy", "lifo").
		Str("level", "high").
		Str("pid", eliminatedPID.String()).
		Done()
	return eliminatedPID
}

// eliminateConnectionsLIFO eliminates connections using the LIFO (Last-In-First-Out) strategy.
// If eliminateHigh is true, it prioritizes eliminating high-level connections.
// It returns the eliminated peer ID and a flag indicating successful elimination.
func (l *LevelConnectionManager) eliminateConnectionsLIFO(eliminateHigh bool) (peer.ID, bool) {
	hCount := len(l.highLevelConn)
	lCount := len(l.lowLevelConn)

	if hCount+lCount > l.maxCountOfPeers {
		if lCount > 0 {
			eliminatedPID := l.closeLowLevelConnLast(lCount)
			return eliminatedPID, true
		}

		if !eliminateHigh {
			return "", false
		}

		eliminatedPID := l.closeHighLevelConnLast(hCount)
		return eliminatedPID, true
	}

	return "", true
}

// eliminateConnections selects the appropriate elimination strategy based on the configured strategy.
// If eliminateHigh is true, it prioritizes eliminating high-level connections.
// It returns the eliminated peer ID and a flag indicating successful elimination.
func (l *LevelConnectionManager) eliminateConnections(eliminateHigh bool) (peer.ID, bool) {
	switch l.strategy {
	case EliminationStrategyRandom:
		return l.eliminateConnectionsRandom(eliminateHigh)
	case EliminationStrategyFIFO:
		return l.eliminateConnectionsFIFO(eliminateHigh)
	case EliminationStrategyLIFO:
		return l.eliminateConnectionsLIFO(eliminateHigh)
	default:
		l.logger.Warn().
			Msg("unknown elimination strategy set, use default.").
			Str("set", l.strategy.String()).
			Str("use", DefaultEliminationStrategy.String()).
			Done()
		l.strategy = DefaultEliminationStrategy
		return l.eliminateConnections(eliminateHigh)
	}
}

// Close closes all the connections managed by LevelConnectionManager.
// It acquires a lock to ensure thread safety.
// It returns an error if there was a problem closing any of the connections, otherwise, it returns nil.
func (l *LevelConnectionManager) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close all low-level connections
	for _, connections := range l.lowLevelConn {
		connections.connSet.Range(func(c network.Connection) bool {
			_ = c.Close()
			return true
		})
	}

	// Close all high-level connections
	for _, connections := range l.highLevelConn {
		connections.connSet.Range(func(c network.Connection) bool {
			_ = c.Close()
			return true
		})
	}

	return nil
}

// AddPeerConnection adds a peer connection to the LevelConnectionManager.
// It acquires a lock to ensure thread safety.
// It returns true if the connection was successfully added, and false otherwise.
func (l *LevelConnectionManager) AddPeerConnection(pid peer.ID, conn network.Connection) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Debug().Msg("add peer connection").Str("pid", pid.String()).Done()

	// Check if the peer is in the high-level level
	highLevel := l.highLevelPeers.Exist(pid)

	var connSet *types.Set[network.Connection]
	if highLevel {
		connSet, _ = l.getHighLevelConnections(pid)
	} else {
		connSet, _ = l.getLowLevelConnections(pid)
	}

	if connSet != nil { // Connection set exists for the peer
		if connSet.Put(conn) {
			return true
		}
		l.logger.Warn().Msg("connection exists, ignored.").Str("pid", pid.String()).Done()
		return false
	}

	// Create a new connection set for the peer
	connSet = types.NewSet[network.Connection]()
	connSet.Put(conn)

	pc := &peerConnections{
		pid:     pid,
		connSet: connSet,
	}

	// Add the peer connection to the appropriate level
	if highLevel {
		l.highLevelConn = append(l.highLevelConn, pc)
	} else {
		l.lowLevelConn = append(l.lowLevelConn, pc)
	}

	// Perform connection elimination if necessary
	ePID, ok := l.eliminateConnections(highLevel)
	if !ok {
		l.logger.Fatal().Msg("eliminate connection failed. This shouldn't happen, it's a bug.")
	}

	// Add the eliminated peer ID to the set of eliminated peers
	if ePID != "" {
		l.eliminatePeers.Put(ePID)
	}

	return true
}

// RemovePeerConnection removes a peer connection from the LevelConnectionManager.
// It acquires a lock to ensure thread safety.
// It returns true if the connection was successfully removed, and false otherwise.
func (l *LevelConnectionManager) RemovePeerConnection(pid peer.ID, conn network.Connection) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	res := false

	// Check if the peer is in the high-level connections
	c, idx := l.getHighLevelConnections(pid)
	if idx != -1 {
		if c.Exist(conn) {
			// Remove the connection from the high-level connections
			if idx == len(l.highLevelConn)-1 {
				l.highLevelConn = l.highLevelConn[:idx]
			} else {
				l.highLevelConn = append(l.highLevelConn[:idx], l.highLevelConn[idx+1:]...)
			}
			res = true
		}
	}

	// Check if the peer is in the low-level connections
	c, idx = l.getLowLevelConnections(pid)
	if idx != -1 {
		if c.Exist(conn) {
			// Remove the connection from the low-level connections
			if idx == len(l.lowLevelConn)-1 {
				l.lowLevelConn = l.lowLevelConn[:idx]
			} else {
				l.lowLevelConn = append(l.lowLevelConn[:idx], l.lowLevelConn[idx+1:]...)
			}
			res = true
		}
	}

	// Remove the peer ID from the set of eliminated peers
	if l.eliminatePeers.Remove(pid) {
		res = true
	}

	return res
}

// ExistPeerConnection checks if a peer connection exists in the LevelConnectionManager.
// It acquires a lock to ensure thread safety.
// It returns true if the connection exists, and false otherwise.
func (l *LevelConnectionManager) ExistPeerConnection(pid peer.ID, conn network.Connection) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	connSet, idx := l.getHighLevelConnections(pid)

	// Check if the peer is in the high-level connections
	if idx == -1 {
		connSet, _ = l.getLowLevelConnections(pid)
	}

	// Check if the connection set exists for the peer
	if connSet != nil {
		return connSet.Exist(conn)
	}

	return false
}

// PeerConnection returns the network.Connection associated with a given peer ID in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
// It returns the network.Connection if it exists, otherwise it returns nil.
func (l *LevelConnectionManager) PeerConnection(pid peer.ID) network.Connection {
	l.mu.RLock()
	defer l.mu.RUnlock()

	connSet, idx := l.getHighLevelConnections(pid)

	// Check if the peer is in the high-level connections
	if idx == -1 {
		connSet, idx = l.getLowLevelConnections(pid)
	}

	var res network.Connection

	// Get the connection from the connection set if it exists
	if idx != -1 {
		connSet.Range(func(conn network.Connection) bool {
			res = conn
			return false
		})
	}

	return res
}

// PeerAllConnection returns all network.Connections associated with a given peer ID in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
// It returns a slice of network.Connections if connections exist, otherwise it returns an empty slice.
func (l *LevelConnectionManager) PeerAllConnection(pid peer.ID) []network.Connection {
	l.mu.RLock()
	defer l.mu.RUnlock()

	connSet, idx := l.getHighLevelConnections(pid)

	// Check if the peer is in the high-level connections
	if idx == -1 {
		connSet, idx = l.getLowLevelConnections(pid)
	}

	var res []network.Connection

	// Get all connections from the connection set if it exists
	if idx != -1 {
		connSet.Range(func(conn network.Connection) bool {
			res = append(res, conn)
			return true
		})
	}

	return res
}

// Connected checks if a given peer ID is connected in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
// It returns true if the peer is connected, and false otherwise.
func (l *LevelConnectionManager) Connected(pid peer.ID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if the peer is in the high-level connections and has at least one connection
	if s, idx := l.getHighLevelConnections(pid); idx != -1 && s.Size() > 0 {
		return true
	}

	// Check if the peer is in the low-level connections and has at least one connection
	if s, idx := l.getLowLevelConnections(pid); idx != -1 && s.Size() > 0 {
		return true
	}

	return false
}

// Allowed checks if a given peer ID is allowed to establish new connections in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
// It returns true if the peer is allowed, and false otherwise.
func (l *LevelConnectionManager) Allowed(pid peer.ID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var currentCount int64

	// Check if the peer is in the high-level connections and increment the current count
	if s, idx := l.getHighLevelConnections(pid); idx != -1 {
		currentCount += s.Size()
	}

	// Check if the peer is in the low-level connections and increment the current count
	if s, idx := l.getLowLevelConnections(pid); idx != -1 {
		currentCount += s.Size()
	}

	// Check if the current count exceeds the maximum allowed connections per peer
	if currentCount > 0 {
		return currentCount < l.maxCountOfConnectionsEachPeer
	}

	highLevel := l.highLevelPeers.Exist(pid)

	if l.strategy == EliminationStrategyLIFO {
		// Check if the strategy is LIFO and the peer is in the high-level connections
		if highLevel {
			return len(l.highLevelConn) < l.maxCountOfPeers
		}
		// Check if the strategy is LIFO and the peer is not in the high-level connections
		return len(l.highLevelConn)+len(l.lowLevelConn) < l.maxCountOfPeers
	}

	// Check if the strategy is not LIFO and the peer is not in the high-level connections
	if !highLevel {
		return len(l.highLevelConn) < l.maxCountOfPeers
	}

	return true
}

// MaxCountOfPeersAllowed returns the maximum count of peers allowed in the LevelConnectionManager.
func (l *LevelConnectionManager) MaxCountOfPeersAllowed() int {
	return l.maxCountOfPeers
}

// CountOfPeers returns the current count of connected peers in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
func (l *LevelConnectionManager) CountOfPeers() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.highLevelConn) + len(l.lowLevelConn)
}

// AllPeers returns a slice containing all the peer IDs of connected peers in the LevelConnectionManager.
// It acquires a read lock to ensure thread safety.
func (l *LevelConnectionManager) AllPeers() []peer.ID {
	l.mu.RLock()
	defer l.mu.RUnlock()

	highCount := len(l.highLevelConn)
	lowCount := len(l.lowLevelConn)
	res := make([]peer.ID, 0, highCount+lowCount)

	// Append peer IDs from high-level connections
	for _, connections := range l.highLevelConn {
		res = append(res, connections.pid)
	}

	// Append peer IDs from low-level connections
	for _, connections := range l.lowLevelConn {
		res = append(res, connections.pid)
	}

	return res
}
