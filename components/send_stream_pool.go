package components

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/safe"
	"github.com/rambollwong/rainbowbee/util"
	"github.com/rambollwong/rainbowlog"
)

var (
	ErrStreamPoolClosed      = errors.New("stream pool closed")
	ErrNoStreamCanBeBorrowed = errors.New("no stream can be borrowed")
	ErrInitSizeTooBig        = errors.New("init size is too big")
	sendStreamPoolSeq        uint64                       // sendStreamPoolSeq is the Sequence of SendStreamPool, which is a local identifier used to distinguish SendStreamPools of different Connections on the same remote peer.
	idlePercentage                                  = 0.2 // idlePercentage is used to control the idle number ratio threshold triggered by expand. If it is lower than the threshold, expand execution will be triggered.
	_                        manager.SendStreamPool = (*SendStreamPool)(nil)
)

// SendStreamPool is a pool of send streams.
type SendStreamPool struct {
	mu                                       sync.RWMutex
	once                                     sync.Once
	conn                                     network.Connection
	initSize, maxSize, idleSize, currentSize int
	poolC                                    chan network.SendStream

	expandingSignalC chan struct{}
	idlePercentage   float64
	closeC           chan struct{}
	seq              uint64

	logger *rainbowlog.Logger
}

// NewSendStreamPool creates a new instance of SendStreamPool.
// It initializes the SendStreamPool with the given initial size, maximum size, connection.
// If the initial size is greater than the maximum size, it returns an error.
func NewSendStreamPool(initSize, maxSize int, conn network.Connection) (*SendStreamPool, error) {
	if initSize > maxSize {
		return nil, ErrInitSizeTooBig
	}
	return &SendStreamPool{
		mu:               sync.RWMutex{},
		once:             sync.Once{},
		conn:             conn,
		initSize:         initSize,
		maxSize:          maxSize,
		idleSize:         0,
		currentSize:      0,
		poolC:            make(chan network.SendStream, maxSize),
		expandingSignalC: make(chan struct{}, 1),
		idlePercentage:   idlePercentage,
		closeC:           make(chan struct{}),
		seq:              atomic.AddUint64(&sendStreamPoolSeq, 1),
		logger: conn.Network().Logger().SubLogger(
			rainbowlog.WithLabels("SEND-STREAM-POOL"),
		),
	}, nil
}

// Close closes the SendStreamPool.
func (s *SendStreamPool) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.close()
}

// Connection returns the network connection associated with the SendStreamPool.
func (s *SendStreamPool) Connection() network.Connection {
	return s.conn
}

// InitStreams initializes the send streams in the SendStreamPool.
// It acquires a lock, opens send streams until the currentSize reaches the initSize,
// logs the initialization, and starts the expandLoop goroutine.
func (s *SendStreamPool) InitStreams() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.once.Do(func() {
		for s.currentSize < s.initSize {
			if err = s.openSendStream(); err != nil {
				return
			}
		}
		s.logger.Debug().
			Msg("send streams initialized.").
			Int("init_size", s.initSize).
			Str("remote_pid", s.conn.RemotePeerID().String()).
			Done()
		safe.LoggerGo(s.logger, s.expandLoop)
	})
	return err
}

// BorrowStream borrows a send stream from the SendStreamPool.
// It acquires a lock, tries to retrieve a send stream from the pool,
// if none is available and the currentSize is not at the maxSize, it opens a new send stream,
// if the currentSize is at the maxSize, it returns ErrNoStreamCanBeBorrowed,
// otherwise, it returns the borrowed send stream.
func (s *SendStreamPool) BorrowStream() (sendStream network.SendStream, err error) {
	s.mu.Lock()
	select {
	case <-s.closeC:
		s.mu.Unlock()
		return nil, ErrStreamPoolClosed
	case sendStream = <-s.poolC:
		s.idleSize--
		s.mu.Unlock()
		defer s.triggerExpand()
		return sendStream, nil
	default:
		if s.currentSize >= s.maxSize {
			s.mu.Unlock()
			return nil, ErrNoStreamCanBeBorrowed
		}
		s.triggerExpand()
		// unlock it here so that expend can execute normally
		s.mu.Unlock()
		s.mu.Lock()
		select {
		case <-s.closeC:
			s.mu.Unlock()
			return nil, ErrStreamPoolClosed
		case sendStream = <-s.poolC:
			s.idleSize--
			s.mu.Unlock()
			return sendStream, nil
		case <-time.After(time.Second): // This is to prevent permanent blocking caused by logical deadlock
			s.mu.Unlock()
			time.Sleep(time.Second)
			return s.BorrowStream()
		}
	}
}

// ReturnStream returns a send stream to the SendStreamPool.
// It acquires a lock, tries to return the send stream to the pool,
// if the pool is full, it closes the send stream, otherwise, it adds the send stream to the pool.
func (s *SendStreamPool) ReturnStream(sendStream network.SendStream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.closeC:
		return ErrStreamPoolClosed
	case s.poolC <- sendStream:
		s.idleSize++
	default:
		_ = sendStream.Close()
	}
	return nil
}

// DropStream closes and drops a send stream from the SendStreamPool.
// This method should be invoked only when errors are found.
// It acquires a lock, closes the send stream, decreases the currentSize,
// and resets the currentSize and idleSize if they become negative.
func (s *SendStreamPool) DropStream(sendStream network.SendStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = sendStream.Close()
	s.currentSize--
	if s.currentSize < 0 {
		s.currentSize = 0
		s.idleSize = 0
	}
}

// MaxSize returns the maximum size of the SendStreamPool.
func (s *SendStreamPool) MaxSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSize
}

// CurrentSize returns the current size of the SendStreamPool.
func (s *SendStreamPool) CurrentSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSize
}

// IdleSize returns the number of idle send streams in the SendStreamPool.
func (s *SendStreamPool) IdleSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.idleSize
}

// close closes the SendStreamPool.
func (s *SendStreamPool) close() error {
	select {
	case <-s.closeC:
	default:
		close(s.closeC)
	}
	return nil
}

// openSendStream opens a new send stream and adds it to the pool.
func (s *SendStreamPool) openSendStream() (err error) {
	var newSendStream network.SendStream
	newSendStream, err = s.conn.OpenSendStream()
	if err != nil {
		return err
	}
	s.poolC <- newSendStream
	s.idleSize++
	s.currentSize++
	return nil
}

// expandingThreshold returns the threshold at which the pool should expand.
func (s *SendStreamPool) expandingThreshold() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int(math.Round(s.idlePercentage * float64(s.currentSize)))
}

// expandCheck checks if the pool should be expanded based on the threshold.
func (s *SendStreamPool) expandCheck() bool {
	threshold := s.expandingThreshold()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return threshold >= s.idleSize && s.currentSize < s.maxSize && s.currentSize != 0
}

// triggerExpand signals the pool to expand by sending a signal to the expandingSignalC channel.
func (s *SendStreamPool) triggerExpand() {
	select {
	case <-s.closeC:
	case s.expandingSignalC <- struct{}{}:
	default:
	}
}

// expandStream expands the pool by opening new send streams.
func (s *SendStreamPool) expandStream() {
	s.mu.Lock()
	defer s.mu.Unlock()
	expandSize := s.initSize / 2
	if expandSize == 0 {
		expandSize = 1
	}
	if expandSize+s.currentSize > s.maxSize {
		expandSize = s.maxSize - s.currentSize
	}
	s.logger.Debug().
		Msg("expanding send streams.").
		Uint64("seq", s.seq).
		Int("expand", expandSize).
		Int("max", s.maxSize).
		Int("current", s.currentSize).
		Str("remote_pid", s.conn.RemotePeerID().String()).
		Done()
	for i := 0; i < expandSize; i++ {
		err := s.openSendStream()
		if util.CheckClosedConnectionWithErr(s.Connection(), err) {
			s.logger.Error().
				Msg("failed to expand send streams, connection closed.").
				Uint64("seq", s.seq).
				Str("remote_pid", s.conn.RemotePeerID().String()).
				Err(err).
				Done()
			_ = s.close()
			return
		}
		s.logger.Error().
			Msg("failed to expand send streams.").
			Uint64("seq", s.seq).
			Str("remote_pid", s.conn.RemotePeerID().String()).
			Err(err).
			Done()
		continue
	}
	s.logger.Info().
		Msg("send streams expanded.").
		Uint64("seq", s.seq).
		Int("expand", expandSize).
		Int("max", s.maxSize).
		Int("current", s.currentSize).
		Str("remote_pid", s.conn.RemotePeerID().String()).
		Done()
	s.triggerExpand()
}

// expandLoop is a goroutine that periodically checks if the pool should be expanded and triggers the expansion.
func (s *SendStreamPool) expandLoop() {
	for {
		select {
		case <-s.closeC:
			return
		case <-s.expandingSignalC:
			if !s.expandCheck() {
				continue
			}
			s.expandStream()
		}
	}
}
