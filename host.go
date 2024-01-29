package rainbowbee

import (
	"bytes"
	"context"
	"errors"
	"sync"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/components"
	"github.com/rambollwong/rainbowbee/core/blacklist"
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/store"
	"github.com/rambollwong/rainbowbee/log"
	"github.com/rambollwong/rainbowbee/peerstore"
	"github.com/rambollwong/rainbowbee/util"
	"github.com/rambollwong/rainbowcat/types"
	catutil "github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"
)

const (
	loggerLabel               = "HOST"
	defaultSupervisorTryTimes = 10
	defaultInitSendStreamSize = 10
	defaultMaxSendStreamSize  = 50
)

var (
	ErrPeerNotConnected       = errors.New("peer is not connected")
	ErrProtocolNotSupported   = errors.New("protocol is not supported")
	ErrSendStreamPoolNotFound = errors.New("send stream pool not found")
	ErrSendStreamWriteFailed  = errors.New("failed to write to send stream")
	ErrWriteMsgIncompletely   = errors.New("write msg incompletely")
	ErrInvalidAddr            = errors.New("invalid address")
	ErrNoAddressFoundInStore  = errors.New("no address found in store")
	ErrAllDialFailed          = errors.New("all dial failed")

	_ host.Host = (*Host)(nil)
)

// hostConfig holds the configuration settings for the Host.
type hostConfig struct {
	ListenAddresses   []ma.Multiaddr           // The listen addresses for the Host.
	DirectPeers       map[peer.ID]ma.Multiaddr // The direct peers with their corresponding multiaddresses.
	BlackNetAddresses []string                 // The blacklisted network addresses.
	BlackPIDs         []peer.ID                // The blacklisted peer IDs.
	CompressMsg       bool                     // Whether to compress messages.
}

// Host is the main struct representing the Rainbowbee host.
type Host struct {
	cfg hostConfig

	once sync.Once
	mu   sync.RWMutex

	ctx context.Context

	// network config
	nwCfg NetworkConfig
	// network
	nw network.Network
	// peer store
	store store.PeerStore

	// managers

	connMgr               manager.ConnectionManager
	supervisor            manager.ConnectionSupervisor
	protocolMgr           manager.ProtocolManager
	protocolExr           manager.ProtocolExchanger
	sendStreamPoolBuilder manager.SendStreamPoolBuilder
	sendStreamPoolMgr     manager.SendStreamPoolManager
	receiveStreamMgr      manager.ReceiveStreamManager

	// black list
	blacklist blacklist.PeerBlackList

	// notifiees
	notifiee *types.Set[host.Notifiee]

	// connExclusive is used to prevent the problem that two new connections may conflict
	// when two peers dial each other.
	connExclusive     map[peer.ID]network.Connection
	connExclusiveLock sync.Mutex

	// chanel

	pushProtocolSignalC chan struct{}
	notifyConnC         chan network.Connection
	closeC              chan struct{}

	// logger
	logger *rainbowlog.Logger
}

// NewHost creates a new host with the provided options.
func NewHost(opts ...Option) (h *Host, err error) {
	// Initialize the host with default values.
	h = &Host{
		cfg:                   hostConfig{},
		once:                  sync.Once{},
		mu:                    sync.RWMutex{},
		ctx:                   context.Background(),
		nw:                    nil,
		store:                 nil,
		connMgr:               nil,
		supervisor:            nil,
		protocolMgr:           nil,
		protocolExr:           nil,
		sendStreamPoolBuilder: nil,
		sendStreamPoolMgr:     nil,
		receiveStreamMgr:      nil,
		blacklist:             nil,
		notifiee:              types.NewSet[host.Notifiee](),
		connExclusive:         make(map[peer.ID]network.Connection),
		connExclusiveLock:     sync.Mutex{},
		pushProtocolSignalC:   make(chan struct{}, 2),
		notifyConnC:           make(chan network.Connection),
		closeC:                make(chan struct{}),
		logger:                log.Logger.SubLogger(rainbowlog.WithLabels(log.DefaultLoggerLabel, loggerLabel)),
	}

	// Apply the provided options to the host.
	if err = h.apply(opts...); err != nil {
		return nil, err
	}

	// Initialize the network.
	h.nw, err = h.nwCfg.NewNetwork()
	if err != nil {
		return nil, err
	}

	// Initialize the peer store if not provided.
	if h.store == nil {
		h.store = peerstore.NewBasicPeerStore(h.ID())
	}
	if c, ok := h.store.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the connection supervisor if not provided.
	if h.supervisor == nil {
		h.supervisor = components.NewConnectionSupervisor(defaultSupervisorTryTimes)
	}
	if c, ok := h.supervisor.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the connection manager if not provided.
	if h.connMgr == nil {
		h.connMgr = components.NewLevelConnectionManager()
	}
	if c, ok := h.connMgr.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the send stream pool builder if not provided.
	if h.sendStreamPoolBuilder == nil {
		h.sendStreamPoolBuilder = func(conn network.Connection) (manager.SendStreamPool, error) {
			return components.NewSendStreamPool(defaultInitSendStreamSize, defaultMaxSendStreamSize, conn)
		}
	}

	// Initialize the send stream pool manager if not provided.
	if h.sendStreamPoolMgr == nil {
		h.sendStreamPoolMgr = components.NewSendStreamPoolManager()
	}
	if c, ok := h.sendStreamPoolMgr.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the receive stream manager if not provided.
	if h.receiveStreamMgr == nil {
		h.receiveStreamMgr = components.NewReceiveStreamManager()
	}
	if c, ok := h.receiveStreamMgr.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the protocol manager if not provided.
	if h.protocolMgr == nil {
		h.protocolMgr = components.NewProtocolManager(h.store)
	}
	if c, ok := h.protocolMgr.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the protocol exchanger if not provided.
	if h.protocolExr == nil {
		h.protocolExr = components.NewProtocolExchanger(h.protocolMgr)
	}
	if c, ok := h.protocolExr.(host.Components); ok {
		c.AttachHost(h)
	}

	// Initialize the blacklist if not provided.
	if h.blacklist == nil {
		h.blacklist = components.NewBlacklist()
	}
	if c, ok := h.blacklist.(host.Components); ok {
		c.AttachHost(h)
	}

	// Add blacklisted IP addresses and ports.
	for _, address := range h.cfg.BlackNetAddresses {
		h.blacklist.AddIPAndPort(address)
	}

	// Add blacklisted peer IDs.
	for _, pid := range h.cfg.BlackPIDs {
		h.blacklist.AddPeerID(pid)
	}

	// Set the protocol supported and unsupported notification handlers.
	if err = h.protocolMgr.SetProtocolSupportedNotifyFunc(h.notifyProtocolSupportedHandlers); err != nil {
		return nil, err
	}
	if err = h.protocolMgr.SetProtocolUnsupportedNotifyFunc(h.notifyProtocolUnsupportedHandlers); err != nil {
		return nil, err
	}

	// Register the protocol exchanger's message payload handler.
	if err = h.RegisterMsgPayloadHandler(h.protocolExr.ProtocolID(), h.protocolExr.Handle); err != nil {
		return nil, err
	}

	// Set the connection handler for new incoming connections.
	h.nw.SetConnHandler(h.handleNewConnection)

	return h, nil
}

// Start starts the host, allowing it to handle incoming requests or messages.
func (h *Host) Start() (err error) {
	// Ensure that the host is started only once using the sync.Once object.
	h.once.Do(func() {
		// Log an informational message indicating that the host is starting.
		h.logger.Info().Msg("starting host...").Done()

		// Create a channel for signaling the host to stop.
		h.closeC = make(chan struct{})

		// Run the host's main loop.
		h.runLoop()

		// Listen on the specified listen addresses.
		err = h.nw.Listen(h.ctx, h.cfg.ListenAddresses...)
		if err != nil {
			return
		}

		// Start the supervisor.
		err = h.supervisor.Start()
		if err != nil {
			// Log an error message if the host failed to start and return the error.
			h.logger.Error().Msg("failed to start host.").Err(err).Done()
			return
		}

		// Log an informational message indicating that the host has started.
		h.logger.Info().Msg("host started.").Done()
	})

	// Return any error that occurred during the start process.
	return err
}

// Stop stops the host, closing connections and stopping necessary components.
func (h *Host) Stop() error {
	// Log an informational message indicating that the host is stopping.
	h.logger.Info().Msg("stopping host...").Done()

	// Reset the sync.Once object to allow starting the host again.
	defer func() { h.once = sync.Once{} }()

	// Close the close channel to signal the host to stop.
	close(h.closeC)

	// Stop the supervisor.
	if err := h.supervisor.Stop(); err != nil {
		return err
	}

	// Close the connection manager.
	if err := h.connMgr.Close(); err != nil {
		return err
	}

	// Close the network.
	if err := h.nw.Close(); err != nil {
		return err
	}

	// Log an informational message indicating that the host has stopped.
	h.logger.Info().Msg("host stopped.").Done()

	return nil
}

// Context returns the context associated with the host.
func (h *Host) Context() context.Context {
	return h.ctx
}

// ID returns the local peer ID of the host.
func (h *Host) ID() peer.ID {
	return h.nw.LocalPeerID()
}

// RegisterMsgPayloadHandler registers a message payload handler for a given protocol ID.
func (h *Host) RegisterMsgPayloadHandler(protocolID protocol.ID, handler handler.MsgPayloadHandler) error {
	// Register the message payload handler with the protocol manager.
	if err := h.protocolMgr.RegisterMsgPayloadHandler(protocolID, handler); err != nil {
		return err
	}

	// Log an informational message indicating the registration of a new message payload handler.
	h.logger.Info().
		Msg("new msg payload handler registered.").
		Str("protocol", protocolID.String()).
		Done()

	// Trigger a protocol push to inform other peers about the new handler.
	h.triggerPushProtocol()

	return nil
}

// UnregisterMsgPayloadHandler unregisters a message payload handler for a given protocol ID.
func (h *Host) UnregisterMsgPayloadHandler(protocolID protocol.ID) error {
	// Unregister the message payload handler from the protocol manager.
	if err := h.protocolMgr.UnregisterMsgPayloadHandler(protocolID); err != nil {
		return err
	}

	// Log an informational message indicating the unregistration of a message payload handler.
	h.logger.Info().
		Msg("msg payload handler unregistered.").
		Str("protocol", protocolID.String()).
		Done()

	// Trigger a protocol push to inform other peers about the handler change.
	h.triggerPushProtocol()

	return nil
}

// SendMsg sends a message payload to a specific peer using the specified protocol ID.
func (h *Host) SendMsg(protocolID protocol.ID, receiverPID peer.ID, msgPayload []byte) (err error) {
	// Check if the receiver peer is connected.
	if !h.connMgr.Connected(receiverPID) {
		return ErrPeerNotConnected
	}

	// Check if the protocol is supported by the receiver peer.
	if !h.protocolMgr.SupportedByPeer(receiverPID, protocolID) {
		return ErrProtocolNotSupported
	}

	// Get the send stream pool for the receiver peer.
	sendStreamPool := h.sendStreamPoolMgr.GetPeerBestConnSendStreamPool(receiverPID)
	if sendStreamPool == nil {
		return ErrSendStreamPoolNotFound
	}

	// Borrow a send stream from the send stream pool.
	sendStream, borrowErr := sendStreamPool.BorrowStream()
	if borrowErr != nil {
		return borrowErr
	}
	defer func() {
		// Return or drop the send stream based on the result of the send operation.
		if errors.Is(err, ErrSendStreamWriteFailed) || errors.Is(err, ErrWriteMsgIncompletely) {
			sendStreamPool.DropStream(sendStream)
		} else {
			_ = sendStreamPool.ReturnStream(sendStream)
		}
	}()

	// Create a payload package with the protocol ID and message payload.
	payloadPkg := protocol.NewPayloadPackage(protocolID, msgPayload, h.payloadCompressType())

	// Marshal the payload package into bytes.
	pkgBz, err := payloadPkg.Marshal()
	if err != nil {
		return err
	}

	// Convert the length of the marshaled package to bytes.
	pkgBzLengthBz := catutil.IntToBytes(len(pkgBz))

	// Create a buffer and write the package length and package bytes to it.
	var buffer bytes.Buffer
	buffer.Write(pkgBzLengthBz)
	buffer.Write(pkgBz)

	// Write the buffer contents to the send stream.
	n, err := sendStream.Write(buffer.Bytes())
	if err != nil {
		// Handle errors during write operation.

		h.logger.Debug().Msg("failed to write data to send stream.").Err(err).Done()

		// Check if the network is closed.
		if h.nw.Closed() {
			return nil
		}

		// Check if the connection is closed.
		if util.CheckClosedConnectionWithErr(sendStream.Conn(), err) {
			return err
		}

		err = ErrSendStreamWriteFailed
		return err
	}

	// Check if the write operation was incomplete.
	if n < len(pkgBz)+8 {
		h.logger.Debug().
			Msg("write data to send stream incompletely.").
			Int("n", n).
			Int("expected", len(pkgBz)+8).
			Done()
		err = ErrWriteMsgIncompletely
		return err
	}

	return nil
}

// Dial initiates a connection to a remote address specified by a multiaddress.
func (h *Host) Dial(remoteAddr ma.Multiaddr) (network.Connection, error) {
	// Extract the network address and peer ID from the multiaddress.
	remoteNetAddr, remotePID := util.SplitAddrToTransportAndPID(remoteAddr)

	// Check if the extracted network address and peer ID are valid.
	if remoteNetAddr == nil && remotePID == "" {
		return nil, ErrInvalidAddr
	}

	// Call the internal dial function to establish the connection.
	return h.dial(remotePID, remoteAddr)
}

// PeerStore returns the peer store associated with the host.
func (h *Host) PeerStore() store.PeerStore {
	return h.store
}

// ConnectionManager returns the connection manager associated with the host.
func (h *Host) ConnectionManager() manager.ConnectionManager {
	return h.connMgr
}

// ProtocolManager returns the protocol manager associated with the host.
func (h *Host) ProtocolManager() manager.ProtocolManager {
	return h.protocolMgr
}

// PeerBlackList returns the peer blacklist associated with the host.
func (h *Host) PeerBlackList() blacklist.PeerBlackList {
	return h.blacklist
}

// PeerProtocols returns a list of peer protocols that match the specified protocol IDs.
func (h *Host) PeerProtocols(protocolIDs []protocol.ID) (peerProtocols []*host.PeerProtocols, err error) {
	peerProtocols = make([]*host.PeerProtocols, 0)

	if len(protocolIDs) == 0 {
		return peerProtocols, nil
	}

	// Get all connected peers.
	allPIDs := h.connMgr.AllPeers()

Loop:
	for _, pid := range allPIDs {
		for _, protocolID := range protocolIDs {
			// Check if the peer supports the protocol.
			if !h.protocolMgr.SupportedByPeer(pid, protocolID) {
				continue Loop
			}
		}

		// Add the peer and its supported protocols to the result list.
		peerProtocols = append(peerProtocols, &host.PeerProtocols{
			PID:       pid,
			Protocols: h.protocolMgr.SupportedProtocolsOfPeer(pid),
		})
	}

	return peerProtocols, nil
}

// PeerSupportProtocol checks if a peer supports a specific protocol.
func (h *Host) PeerSupportProtocol(pid peer.ID, protocolID protocol.ID) bool {
	return h.protocolMgr.SupportedByPeer(pid, protocolID)
}

// Notify adds a notifiee to the host, allowing it to receive notifications.
func (h *Host) Notify(notifiee host.Notifiee) {
	h.notifiee.Put(notifiee)
}

// AddDirectPeer adds a direct peer to the host.
func (h *Host) AddDirectPeer(dp ma.Multiaddr) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Extract the peer ID from the multiaddress.
	_, pid := util.SplitAddrToTransportAndPID(dp)

	// Check if the extracted peer ID is valid.
	if len(pid) == 0 {
		return nil
	}

	// Initialize the directPeers map if it is nil.
	if h.cfg.DirectPeers == nil {
		h.cfg.DirectPeers = make(map[peer.ID]ma.Multiaddr)
	}

	// Add the peer ID and multiaddress to the directPeers map.
	h.cfg.DirectPeers[pid] = dp

	// Set the peer's address in the supervisor.
	h.supervisor.SetPeerAddr(pid, dp)

	// Add the peer to the high-level connection manager if applicable.
	if levelConnManager, ok := h.connMgr.(*components.LevelConnectionManager); ok {
		levelConnManager.AddHighLevelPeer(pid)
	}

	return nil
}

// ClearDirectPeers removes all direct peers from the host.
func (h *Host) ClearDirectPeers() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clear the directPeers map.
	h.cfg.DirectPeers = make(map[peer.ID]ma.Multiaddr)

	// Remove all peers from the supervisor.
	h.supervisor.RemoveAllPeer()
}

// LocalAddresses returns the list of multiaddresses the host is listening on.
func (h *Host) LocalAddresses() []ma.Multiaddr {
	return h.nw.ListenAddresses()
}

// Network returns the network associated with the host.
func (h *Host) Network() network.Network {
	return h.nw
}

// notifyProtocolHandlers notifies all registered notifiees about the support or lack of support for a specific protocol by a peer.
func (h *Host) notifyProtocolHandlers(protocolID protocol.ID, pid peer.ID, connectedOrDisconnected bool) {
	h.notifiee.Range(func(n host.Notifiee) bool {
		if connectedOrDisconnected {
			n.OnPeerProtocolSupported(protocolID, pid)
		} else {
			n.OnPeerProtocolUnsupported(protocolID, pid)
		}
		return true
	})
}

// notifyProtocolSupportedHandlers notifies all registered notifiees that the given protocol is supported by the peer.
func (h *Host) notifyProtocolSupportedHandlers(protocolID protocol.ID, pid peer.ID) {
	h.notifyProtocolHandlers(protocolID, pid, true)
}

// notifyProtocolUnsupportedHandlers notifies all registered notifiees that the given protocol is unsupported by the peer.
func (h *Host) notifyProtocolUnsupportedHandlers(protocolID protocol.ID, pid peer.ID) {
	h.notifyProtocolHandlers(protocolID, pid, false)
}

func (h *Host) notifyConnectHandlers(pid peer.ID, connectedOrDisconnected bool) {
	h.notifiee.Range(func(n host.Notifiee) bool {
		if connectedOrDisconnected {
			n.OnPeerConnected(pid)
		} else {
			n.OnPeerDisconnected(pid)
		}
		return true
	})
}

// loop is the main event loop of the host, responsible for handling connection notifications and closing the loop when requested.
func (h *Host) loop() {
Loop:
	for {
		select {
		case <-h.closeC:
			break Loop
		case conn := <-h.notifyConnC:
			h.notifyConnectHandlers(conn.RemotePeerID(), !conn.Closed())
		}
	}
}

// pushProtocolsToOthers pushes the host's supported protocols to all connected peers.
func (h *Host) pushProtocolsToOthers() {
	others := h.connMgr.AllPeers()
	var wg sync.WaitGroup
	wg.Add(len(others))
	for _, other := range others {
		other := other
		go func(pid peer.ID) {
			defer wg.Done()
			if err := h.protocolExr.PushProtocols(pid); err != nil {
				h.logger.Warn().
					Msg("failed to push protocols.").
					Str("remote_pid", pid.String()).
					Err(err).
					Done()
			}
		}(other)
	}
	wg.Wait()
}

// pushProtocolSignalLoop listens for push protocol signals and triggers the pushProtocolsToOthers function when a signal is received.
func (h *Host) pushProtocolSignalLoop() {
Loop:
	for {
		select {
		case <-h.closeC:
			break Loop
		case <-h.pushProtocolSignalC:
			h.pushProtocolsToOthers()
		}
	}
}

// runLoop starts the main event loops for handling connection notifications and push protocol signals.
func (h *Host) runLoop() {
	go h.loop()
	go h.pushProtocolSignalLoop()
}

// connExclusiveCheck performs an exclusive check for a network connection.
// It takes a network.Connection object conn representing the new connection.
// It returns a boolean value indicating whether the new connection should be kept or dropped.
func (h *Host) connExclusiveCheck(conn network.Connection) bool {
	// Get the remote peer ID of the connection.
	rPID := conn.RemotePeerID()

	// Acquire a lock to ensure exclusive access to the connExclusive map.
	h.connExclusiveLock.Lock()
	defer h.connExclusiveLock.Unlock()

	// Check if there is an existing connection for the remote peer ID.
	oldConn, loaded := h.connExclusive[rPID]
	if loaded {
		// If there is an existing connection, compare the directions of the old and new connections.
		if oldConn.Direction() != conn.Direction() {
			// If the direction of the old connection is different from the direction of the new connection,
			// use the peerID weight to decide which connection to keep.
			var drop network.Direction
			saveMe := conn.LocalPeerID().WeightCompare(conn.RemotePeerID())
			if saveMe {
				// Drop inbound connection, keep outbound connection.
				drop = network.Inbound
			} else {
				// Drop outbound connection, keep inbound connection.
				drop = network.Outbound
			}

			// Perform the appropriate action based on the drop decision.
			switch drop {
			case oldConn.Direction():
				// Keep the new connection and close the old connection.
				h.connExclusive[rPID] = conn
				_ = oldConn.Close()
			case conn.Direction():
				// Close the new connection and return false to indicate it should be dropped.
				_ = conn.Close()
				return false
			}
		} else {
			// If the connection directions are the same, keep the old connection and close the new connection.
			_ = conn.Close()
			return false
		}
	} else {
		// If there is no existing connection, store the new connection in the connExclusive map.
		h.connExclusive[rPID] = conn
	}

	// Return true to indicate that the new connection should be kept.
	return true
}

// removeConnExclusive removes the exclusive connection entry for the specified peer ID from the host's exclusive connection map.
func (h *Host) removeConnExclusive(pid peer.ID) {
	h.connExclusiveLock.Lock()
	defer h.connExclusiveLock.Unlock()
	delete(h.connExclusive, pid)
}

// handleClosingConn is responsible for handling the closing of a connection.
func (h *Host) handleClosingConn(conn network.Connection) {
	// Close the connection.
	_ = conn.Close()

	// Get the remote peer ID of the connection.
	rPID := conn.RemotePeerID()

	// Remove the connection from the connection manager.
	if !h.connMgr.RemovePeerConnection(rPID, conn) {
		// If the connection is not successfully removed, return.
		return
	}

	h.logger.Info().Msg("a connection disestablished.").
		Str("remote_pid", conn.RemotePeerID().String()).
		Str("remote_addr", conn.RemoteAddr().String()).
		Str("direction", conn.Direction().String()).
		Done()

	// Check if the remote peer is disconnected.
	if !h.connMgr.Connected(rPID) {
		h.logger.Info().Msg("peer disconnected.").
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Done()

		// Notify the connection channel about the disconnection.
		h.notifyConnC <- conn

		// Clean up protocols associated with the disconnected peer.
		if err := h.protocolMgr.CleanPeer(rPID); err != nil {
			h.logger.Error().Msg("failed to clean protocol from manager.").
				Str("remote_pid", rPID.String()).
				Err(err).
				Done()
		}
	}

	// Remove the address associated with the connection from the peer store.
	if err := h.store.RemoveAddress(rPID, conn.RemoteAddr()); err != nil {
		h.logger.Error().Msg("failed to remove address from peer store.").
			Str("remote_pid", rPID.String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
	}

	// Remove the connection and close the associated send stream pool.
	if err := h.sendStreamPoolMgr.RemovePeerConnAndCloseSendStreamPool(rPID, conn); err != nil {
		h.logger.Error().Msg("failed to clean all send stream of connection from send stream pool manager.").
			Str("remote_pid", rPID.String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
	}

	// Close all receive streams associated with the connection.
	if err := h.receiveStreamMgr.ClosePeerReceiveStreams(rPID, conn); err != nil {
		h.logger.Error().Msg("failed to clean all receive stream of connection from receive stream manager.").
			Str("remote_pid", rPID.String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
	}
}

// handleReceiveStreamData processes received stream data.
func (h *Host) handleReceiveStreamData(dataBz []byte, pid peer.ID) {
	// If the received data is empty, return without further processing.
	if len(dataBz) == 0 {
		return
	}

	// Create an empty PayloadPkg struct to unmarshal the received data.
	pkg := &protocol.PayloadPkg{}

	// Unmarshal the received data into the PayloadPkg struct.
	if err := pkg.Unmarshal(dataBz); err != nil {
		// If unmarshaling fails, log a warning message and drop the payload.
		h.logger.Warn().
			Msg("failed to unmarshal payload package, drop it.").
			Str("remote_pid", pid.String()).
			Err(err).
			Done()
		return
	}

	// Retrieve the payload handler associated with the protocol ID of the payload package.
	payloadHandler := h.protocolMgr.Handler(pkg.ProtocolID())

	// If the payload handler is not found (nil), log a warning message and drop the payload.
	if payloadHandler == nil {
		h.logger.Warn().
			Msg("payload handler not found, drop the payload.").
			Str("protocol", pkg.ProtocolID().String()).
			Str("remote_pid", pid.String()).
			Done()
	}
}

// receiveStreamHandlerLoop is a loop that handles the receive stream.
func (h *Host) receiveStreamHandlerLoop(receiveStream network.ReceiveStream) {
	conn := receiveStream.Conn()
	var err error

Loop:
	for {
		// Check if the receive stream's connection is closed.
		if conn.Closed() {
			// Call the handleClosingConn method to handle the closing connection.
			h.handleClosingConn(conn)
			// Break the loop.
			break Loop
		}

		// Read the length of the incoming data package.
		dataLength, _, e := util.ReadPackageLength(receiveStream)
		if e != nil {
			// If reading the package length encounters an error, assign it to the 'err' variable and break the loop.
			err = e
			break Loop
		}

		// Read the data package.
		dataBz, e := util.ReadPackageData(receiveStream, dataLength)
		if e != nil {
			// If reading the data encounters an error, assign it to the 'err' variable and break the loop.
			err = e
			break Loop
		}

		h.handleReceiveStreamData(dataBz, conn.RemotePeerID())
	}

	// Handle errors that occurred during the loop.
	if err != nil {
		// Check if the network is closed and return if it is.
		if h.nw.Closed() {
			return
		}

		// Check if the receive stream's connection is closed with the given error and return if it is.
		if util.CheckClosedConnectionWithErr(conn, err) {
			_ = conn.Close()
			return
		}

		// Close the receive stream.
		_ = receiveStream.Close()

		// Remove the receive stream from the receive stream manager.
		if err := h.receiveStreamMgr.RemovePeerReceiveStream(conn, receiveStream); err != nil {
			h.logger.Debug().
				Msg("failed to remove receive stream from the manager.").
				Str("remote_pid", conn.RemotePeerID().String()).
				Str("remote_addr", conn.RemoteAddr().String()).
				Err(err).
				Done()
			return
		}

		// Log a warning message about the failure to read data from the receive stream.
		h.logger.Warn().
			Msg("failed to read data from the receive stream.").
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()

		// If all Receive Streams for a connection are removed, we consider the connection to be disconnected
		if h.receiveStreamMgr.GetConnReceiveStreamCount(conn) == 0 {
			_ = conn.Close()
		}
	}
}

// handleReceiveStream is responsible for handling a receive stream.
func (h *Host) handleReceiveStream(receiveStream network.ReceiveStream) {
	// Add the receive stream to the receive stream manager.
	if err := h.receiveStreamMgr.AddPeerReceiveStream(receiveStream.Conn(), receiveStream); err != nil {
		h.logger.Error().Msg("failed to add receive stream to the manager.").
			Str("remote_pid", receiveStream.Conn().RemotePeerID().String()).
			Str("remote_addr", receiveStream.Conn().RemoteAddr().String()).
			Err(err).
			Done()
		// Close the receive stream.
		_ = receiveStream.Close()
	}

	// Start the receive stream handler loop in a goroutine.
	go h.receiveStreamHandlerLoop(receiveStream)
}

// acceptReceiveStreamLoop is a loop that accepts receive streams on the given network connection.
func (h *Host) acceptReceiveStreamLoop(conn network.Connection) {
Loop:
	for {
		select {
		case <-h.closeC:
			// If the host is closed, break the loop.
			break Loop
		default:
			if conn.Closed() {
				// If the connection is closed, call the handleClosingConn method.
				h.handleClosingConn(conn)
			}
		}

		// Accept a receive stream from the connection.
		receiveStream, err := conn.AcceptReceiveStream()
		if err != nil {
			if util.CheckClosedConnectionWithErr(conn, err) || conn.Closed() {
				// If the connection is closed or the network is closed, break the loop.
				break Loop
			}
			// Log a warning message about the failure to accept a receive stream.
			h.logger.Warn().
				Msg("failed to accept a receive stream").
				Str("remote_addr", conn.RemoteAddr().String()).
				Err(err).
				Done()
			continue
		}

		// Handle the received stream by calling the handleReceiveStream method.
		h.handleReceiveStream(receiveStream)
	}
}

// handleNewConnection handles a new network connection.
// It performs various operations on the connection such as blacklisting, exclusive check, connection manager checks,
// protocol exchange, stream pool initialization, adding the connection to managers, and notifying about the new connection.
// It returns a boolean indicating whether the connection was successfully handled and an error, if any.
func (h *Host) handleNewConnection(conn network.Connection) (bool, error) {
	// Retrieve the remote peer ID from the connection.
	rPID := conn.RemotePeerID()

	// Check if the connection is blacklisted.
	if h.blacklist.BlackConn(conn) {
		h.logger.Info().Msg("new connection is blacklisted.").
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Done()
		return false, nil
	}

	// Perform exclusive check on the connection.
	if !h.connExclusiveCheck(conn) {
		return false, nil
	}
	defer h.removeConnExclusive(rPID)

	// Check if the connection is allowed by the connection manager.
	if !h.connMgr.Allowed(rPID) {
		h.logger.Info().
			Msg("new connection is rejected by connection manager, close it.").
			Str("remote_pid", rPID.String()).
			Done()
		return false, nil
	}

	var (
		rProtocols        []protocol.ID
		err               error
		exchangeProtocol  bool
		myProtocolsLength int
	)

	if !h.connMgr.Connected(rPID) {
		// If it is the first time establishing connection with us, exchange supported protocols.
		myProtocolsLength = len(h.protocolMgr.SupportedProtocolsOfPeer(h.ID()))
		rProtocols, err = h.protocolExr.ExchangeProtocol(conn)
		if err != nil {
			h.logger.Warn().
				Msg("failed to exchange protocols, close the connection.").
				Str("local_pid", conn.LocalPeerID().String()).
				Str("remote_pid", conn.RemotePeerID().String()).
				Err(err).
				Done()
			_ = conn.Close()
			return false, err
		}
		h.logger.Info().Msg("protocols exchanged.").
			Str("local_pid", conn.LocalPeerID().String()).
			Str("remote_pid", conn.RemotePeerID().String()).
			Strs("protocols", catutil.SliceTransformType(rProtocols, func(_ int, item protocol.ID) string {
				return item.String()
			})...).
			Done()
		exchangeProtocol = true
	}

	streamPool, err := h.sendStreamPoolBuilder(conn)
	if err != nil {
		panic("failed to call send stream pool builder")
	}

	err = streamPool.InitStreams()
	if err != nil {
		h.logger.Warn().
			Msg("failed to initialize streams, close the connection.").
			Str("local_pid", conn.LocalPeerID().String()).
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
		_ = conn.Close()
		return false, err
	}

	err = h.sendStreamPoolMgr.AddPeerConnSendStreamPool(rPID, conn, streamPool)
	if err != nil {
		h.logger.Error().
			Msg("failed to add send stream pool to the manager, close the connection.").
			Str("local_pid", conn.LocalPeerID().String()).
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
		_ = conn.Close()
		return false, err
	}

	if !h.connMgr.AddPeerConnection(rPID, conn) {
		h.logger.Debug().Msg("failed to add connection to the manager, close the connection.").
			Str("local_pid", conn.LocalPeerID().String()).
			Str("remote_pid", conn.RemotePeerID().String()).
			Done()
		_ = conn.Close()
		return false, err
	}

	if exchangeProtocol {
		err = h.protocolMgr.SetSupportedProtocolsOfPeer(rPID, rProtocols)
		if err != nil {
			h.logger.Debug().Msg("failed to set supported protocols to the manager, close the connection.").
				Str("local_pid", conn.LocalPeerID().String()).
				Str("remote_pid", conn.RemotePeerID().String()).
				Done()
			_ = conn.Close()
			return false, err
		}
		if myProtocolsLength != len(h.protocolMgr.SupportedProtocolsOfPeer(h.ID())) {
			go func() {
				err := h.protocolExr.PushProtocols(rPID)
				h.logger.Info().Msg("re_push protocols to other").Str("other", rPID.String()).Err(err).Done()
			}()
		}
	}

	go h.acceptReceiveStreamLoop(conn)

	// todo go h.acceptBidirectionalStreamLoop(conn)

	if err = h.store.AddAddress(rPID, conn.RemoteAddr()); err != nil {
		h.logger.Debug().
			Msg("failed to add address to the peer store, close the connection.").
			Str("local_pid", conn.LocalPeerID().String()).
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Err(err).
			Done()
		_ = conn.Close()
		return false, err
	}

	h.logger.Info().Msg("new connection established.").
		Str("remote_pid", conn.RemotePeerID().String()).
		Str("remote_addr", conn.RemoteAddr().String()).
		Str("direction", conn.Direction().String()).
		Done()

	if exchangeProtocol {
		// This is the first time the other party connects.
		h.logger.Info().Msg("peer connected.").
			Str("remote_pid", conn.RemotePeerID().String()).
			Str("remote_addr", conn.RemoteAddr().String()).
			Done()
		h.notifyConnC <- conn
	}

	return true, nil
}

// triggerPushProtocol triggers the push protocol mechanism.
func (h *Host) triggerPushProtocol() {
	select {
	case h.pushProtocolSignalC <- struct{}{}:
	default:

	}
}

func (h *Host) payloadCompressType() protocol.CompressType {
	if h.cfg.CompressMsg {
		return protocol.CompressTypeGzip
	}
	return protocol.CompressTypeNone
}

// dial establishes a network connection to a remote peer identified by pid and addr.
// If addr is nil, it retrieves the remote addresses associated with the peer from the store
// and attempts to dial each address until a successful connection is established.
// If addr is not nil, it directly attempts to dial the provided address.
// Returns the network connection if successful, or an error if all dial attempts fail.
func (h *Host) dial(pid peer.ID, addr ma.Multiaddr) (network.Connection, error) {
	if addr == nil {
		// No specific address provided, retrieve remote addresses from the store.
		remoteAddresses := h.store.GetAddresses(pid)
		if len(remoteAddresses) == 0 {
			h.logger.Debug().Msg("no address found in store.").Str("pid", pid.String()).Done()
			return nil, ErrNoAddressFoundInStore
		}

		// Iterate over each remote address and attempt to dial.
		for _, address := range remoteAddresses {
			tmpAddr, tmpPID := util.SplitAddrToTransportAndPID(address)
			if tmpPID == "" {
				// If the address doesn't include a peer ID, create a new address with the provided peer ID.
				address = util.PIDAndNetAddrToMultiAddr(pid, tmpAddr)
			} else if tmpPID != pid {
				// Skip this address if the peer ID doesn't match the one provided.
				continue
			}
			addr = address
			h.logger.Info().
				Msg("trying to dial.").
				Str("remote_pid", pid.String()).
				Str("remote_addr", addr.String()).
				Done()
			conn, err := h.nw.Dial(h.ctx, addr)
			if err != nil {
				h.logger.Info().Msg("failed to dial.").
					Str("remote_pid", pid.String()).
					Str("remote_addr", addr.String()).
					Err(err).
					Done()
				continue
			}
			return conn, nil
		}
	} else {
		// Dial the provided address directly.
		h.logger.Info().
			Msg("trying to dial.").
			Str("remote_pid", pid.String()).
			Str("remote_addr", addr.String()).
			Done()
		conn, err := h.nw.Dial(h.ctx, addr)
		if err == nil {
			return conn, nil
		}
	}

	// All dial attempts failed.
	h.logger.Info().Msg("all dial failed.").
		Str("remote_pid", pid.String()).
		Str("remote_addr", addr.String()).
		Done()
	return nil, ErrAllDialFailed
}
