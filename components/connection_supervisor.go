package components

import (
	"sync"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/safe"
	"github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"
)

const DefaultTryTimes = 10

var _ manager.ConnectionSupervisor = (*ConnectionSupervisor)(nil)

// ConnectionSupervisor is responsible for managing the connections to necessary peers.
// It implements the manager.ConnectionSupervisor interface.
type ConnectionSupervisor struct {
	mu   sync.RWMutex
	once sync.Once
	host host.Host

	necessaryPeer map[peer.ID]ma.Multiaddr

	watcherTicker *time.Ticker
	signalC       chan struct{}
	closeC        chan struct{}

	tryTimes  int
	actuators map[peer.ID]*dialActuator

	allConnected bool

	hostNotifiee *host.NotifieeBundle

	logger *rainbowlog.Logger
}

// NewConnectionSupervisor creates a new instance of ConnectionSupervisor.
func NewConnectionSupervisor(tryTimes int) *ConnectionSupervisor {
	// If tryTimes is less than or equal to 0, set it to the default value.
	if tryTimes <= 0 {
		tryTimes = DefaultTryTimes
	}
	cs := &ConnectionSupervisor{
		mu:            sync.RWMutex{},
		once:          sync.Once{},
		host:          nil,
		necessaryPeer: make(map[peer.ID]ma.Multiaddr),
		watcherTicker: nil,
		signalC:       make(chan struct{}, 1),
		closeC:        make(chan struct{}),
		tryTimes:      tryTimes,
		actuators:     make(map[peer.ID]*dialActuator),
		allConnected:  false,
		hostNotifiee:  &host.NotifieeBundle{},
		logger:        nil,
	}
	cs.hostNotifiee.OnPeerDisconnectedFunc = cs.NoticePeerDisconnected
	return cs
}

func (c *ConnectionSupervisor) AttachHost(h host.Host) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.host = h
	c.logger = h.Logger().SubLogger(rainbowlog.WithLabels("CONN-SPV"))
	c.host.Notify(c.hostNotifiee)
}

// checkConn checks the connection status of the necessary peers and performs dialing if needed.
func (c *ConnectionSupervisor) checkConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// If there are no necessary peers, return.
	if len(c.necessaryPeer) == 0 || c.allConnected {
		return
	}
	currentConnected := 0
	for pid, addr := range c.necessaryPeer {
		// Skip the host itself and already connected peers.
		if pid == c.host.ID() {
			continue
		}
		if c.host.ConnectionManager().Connected(pid) {
			currentConnected++
			act, ok := c.actuators[pid]
			if ok {
				act.SetFinished()
				delete(c.actuators, pid)
			}
			continue
		}
		c.allConnected = false
		act, ok := c.actuators[pid]
		if !ok {
			act = newDialActuator(pid, addr, c, c.tryTimes)
			c.actuators[pid] = act
		}

		// Skip if it has been given up.
		if act.GiveUp() {
			continue
		}

		// Reset the actuator if it has finished.
		if act.Finished() {
			act.Reset()
		}
		safe.LoggerGo(c.logger, act.Run)
	}
	if currentConnected == len(c.necessaryPeer) {
		c.allConnected = true
		c.logger.Info().Msg("all necessary peers connected.").Done()
	}
}

// loop is the main loop of the ConnectionSupervisor.
func (c *ConnectionSupervisor) loop() {
	c.watcherTicker = time.NewTicker(5 * time.Second)
	for {
		select {
		case <-c.closeC:
			return
		case <-c.signalC:
			c.checkConn()
		case <-c.watcherTicker.C:
			select {
			case c.signalC <- struct{}{}:
			default:
			}
		}
	}
}

// Start starts the ConnectionSupervisor.
func (c *ConnectionSupervisor) Start() error {
	c.once.Do(func() {
		c.closeC = make(chan struct{})
		safe.LoggerGo(c.logger, c.loop)
		c.signalC <- struct{}{}
	})
	return nil
}

// Stop stops the ConnectionSupervisor.
func (c *ConnectionSupervisor) Stop() error {
	close(c.closeC)
	c.once = sync.Once{}
	return nil
}

// SetPeerAddr sets the Multiaddr of a necessary peer.
func (c *ConnectionSupervisor) SetPeerAddr(pid peer.ID, addr ma.Multiaddr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.necessaryPeer[pid] = addr
	select {
	case c.signalC <- struct{}{}:
	default:
	}
}

// RemovePeerAddr removes the Multiaddr of a necessary peer.
func (c *ConnectionSupervisor) RemovePeerAddr(pid peer.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.necessaryPeer, pid)
	delete(c.actuators, pid)
}

// RemoveAllPeer removes all necessary peers.
func (c *ConnectionSupervisor) RemoveAllPeer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.necessaryPeer = make(map[peer.ID]ma.Multiaddr)
}

// NoticePeerDisconnected is called when a necessary peer is disconnected.
func (c *ConnectionSupervisor) NoticePeerDisconnected(pid peer.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.necessaryPeer[pid]; !ok {
		return
	}
	c.allConnected = false
	select {
	case <-c.closeC:
	case c.signalC <- struct{}{}:
	default:
	}
}

type dialActuator struct {
	mu        sync.RWMutex
	pid       peer.ID
	peerAddr  ma.Multiaddr
	fibonacci []int64
	tryTimes  int
	giveUp    bool
	finished  bool
	stateC    chan struct{}

	supervisor *ConnectionSupervisor
}

// newDialActuator creates a new instance of dialActuator.
func newDialActuator(pid peer.ID, addr ma.Multiaddr, supervisor *ConnectionSupervisor, tryTimes int) *dialActuator {
	return &dialActuator{
		pid:        pid,
		peerAddr:   addr,
		fibonacci:  util.FibonacciArray(tryTimes),
		tryTimes:   0,
		giveUp:     false,
		finished:   false,
		stateC:     make(chan struct{}, 1),
		supervisor: supervisor,
	}
}

func (a *dialActuator) GiveUp() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.giveUp
}

func (a *dialActuator) Finished() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.finished
}

// Reset resets the state of the dialActuator.
func (a *dialActuator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tryTimes = 0
	a.giveUp = false
	a.finished = false
}

func (a *dialActuator) SetFinished() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.finished = true
}

func (a *dialActuator) SetUnfinished() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.finished = false
}

func (a *dialActuator) SetGiveUp() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.giveUp = true
}

// Run executes the dialing process to establish a connection with the peer.
func (a *dialActuator) Run() {
	select {
	case a.stateC <- struct{}{}:
		defer func() { <-a.stateC }()
	default:
		return
	}
	if a.GiveUp() || a.Finished() {
		return
	}

	// breakCheck checks for termination conditions during the dialing process.
	breakCheck := func() bool {
		select {
		case <-a.supervisor.closeC:
			return true
		default:
		}

		if a.supervisor.host.ConnectionManager().Connected(a.pid) {
			a.supervisor.logger.Debug().
				Msg("peer connected, dial actuator will exit.").
				Str("pid", a.pid.String()).
				Done()
			a.SetFinished()
			return true
		}

		return false
	}

Loop:
	for {
		if breakCheck() {
			break Loop
		}
		a.supervisor.logger.Debug().
			Msg("dialing to peer").
			Str("pid", a.pid.String()).
			Str("addr", a.peerAddr.String()).
			Done()
		conn, err := a.supervisor.host.Dial(a.peerAddr)
		if breakCheck() {
			break Loop
		}
		if err != nil {
			a.tryTimes++
			a.supervisor.logger.Warn().
				Msg("dial to peer failed.").
				Str("pid", a.pid.String()).
				Str("addr", a.peerAddr.String()).
				Int("times", a.tryTimes).
				Err(err).
				Done()
		} else {
			if conn == nil {
				a.tryTimes++
				a.supervisor.logger.Warn().
					Msg("dial to peer failed, nil connection.").
					Str("pid", a.pid.String()).
					Str("addr", a.peerAddr.String()).
					Int("times", a.tryTimes).
					Done()
			} else {
				if a.pid != conn.RemotePeerID() {
					a.supervisor.logger.Warn().
						Msg("dial to peer failed, pid mismatch, close the connection and give it up.").
						Str("pid", a.pid.String()).
						Str("got", conn.RemotePeerID().String()).
						Err(err).
						Done()
					_ = conn.Close()
					a.SetGiveUp()
					break Loop
				}
				a.supervisor.logger.Debug().
					Msg("dial to peer success.").
					Str("pid", a.pid.String()).
					Str("addr", a.peerAddr.String()).
					Done()
				a.SetFinished()
				break Loop
			}
		}
		if !a.Finished() && !a.GiveUp() {
			if a.tryTimes >= len(a.fibonacci) {
				a.supervisor.logger.Warn().Msg("can not dial to peer, give it up.").
					Str("pid", a.pid.String()).
					Str("addr", a.peerAddr.String()).
					Done()
				a.SetGiveUp()
				break Loop
			}
			timeout := time.Duration(a.fibonacci[a.tryTimes]) * time.Second
			time.Sleep(timeout)
		}
	}
}
