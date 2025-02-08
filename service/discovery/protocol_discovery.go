package discovery

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/discovery"
	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/safe"
	"github.com/rambollwong/rainbowbee/service/discovery/pb"
	"github.com/rambollwong/rainbowbee/util"
	"github.com/rambollwong/rainbowcat/types"
	catutil "github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"
	"google.golang.org/protobuf/proto"
)

const (
	protocolDiscoveryPIDPrefix        = "/protocol-discovery/v0.0.1/"
	DefaultSearchInterval             = time.Minute
	DefaultSearchMaxPeerSize          = 10
	DefaultSearchAsksForNumberOfPeers = 3
)

var (
	_ discovery.Discovery = (*ProtocolDiscovery)(nil)

	ErrSearching = errors.New("already searching")
)

type ProtocolDiscoveryOption func(pd *ProtocolDiscovery)

// WithMaxSearchSize sets the maximum number of peers to search for.
func WithMaxSearchSize(max int) ProtocolDiscoveryOption {
	return func(pd *ProtocolDiscovery) {
		pd.maxSearchSize = max
	}
}

// WithSearchAsksForNumberOfPeers sets the number of peers to ask for during a search.
func WithSearchAsksForNumberOfPeers(num int) ProtocolDiscoveryOption {
	return func(pd *ProtocolDiscovery) {
		pd.asksForPeers = num
	}
}

// WithSearchTimeout sets the timeout for a search operation.
func WithSearchTimeout(timeout time.Duration) ProtocolDiscoveryOption {
	return func(pd *ProtocolDiscovery) {
		pd.searchTimeout = timeout
	}
}

// WithSearchInterval sets the interval between consecutive search operations.
func WithSearchInterval(interval time.Duration) ProtocolDiscoveryOption {
	return func(pd *ProtocolDiscovery) {
		pd.searchInterval = interval
	}
}

// ProtocolDiscovery is a struct that represents a protocol discovery mechanism.
// It is responsible for discovering peers supporting specific protocols and initiating search requests.
type ProtocolDiscovery struct {
	h host.Host

	svcS       *types.Set[string]             // Set of service names to be discovered
	mu         sync.RWMutex                   // Mutex for concurrent access to the searching map
	searchingM map[string]chan []ma.Multiaddr // Map of service names to channels for receiving discovered addresses

	maxSearchSize, asksForPeers   int           // Maximum search size and number of peers to ask for
	searchTimeout, searchInterval time.Duration // Search timeout and interval between search requests

	logger *rainbowlog.Logger // Logger for logging messages and errors
}

// NewProtocolDiscovery creates a new instance of ProtocolDiscovery.
func NewProtocolDiscovery(h host.Host, opts ...ProtocolDiscoveryOption) *ProtocolDiscovery {
	pd := &ProtocolDiscovery{
		h:              h,
		svcS:           types.NewSet[string](),
		mu:             sync.RWMutex{},
		searchingM:     make(map[string]chan []ma.Multiaddr),
		maxSearchSize:  DefaultSearchMaxPeerSize,
		asksForPeers:   DefaultSearchAsksForNumberOfPeers,
		searchTimeout:  -1,
		searchInterval: DefaultSearchInterval,
		logger:         h.Logger().SubLogger(rainbowlog.WithLabels("PROTOCOL-DISC")),
	}

	pd.applyOptions(opts...)

	return pd
}

func (p *ProtocolDiscovery) Announce(ctx context.Context, serviceName string) error {
	serviceName = strings.TrimSpace(serviceName)
	// create protocol ID
	protocolID := p.createProtocolIDWithServiceName(serviceName)
	if !p.svcS.Exist(serviceName) {
		// create protocol msg handler
		h := p.createProtocolMsgHandler(serviceName, protocolID)
		// register discovery protocol to host
		err := p.h.RegisterMsgPayloadHandler(protocolID, h)
		if err != nil {
			return err
		}
		p.svcS.Put(serviceName)
	}

	// send announce msg to all peers who support the protocol
	allPeers, err := p.h.PeerProtocols([]protocol.ID{protocolID})
	if err != nil {
		return err
	}
	if len(allPeers) == 0 {
		return nil
	}
	localAddresses := p.h.LocalAddresses()
	localAddresses = util.ExcludeUnspecifiedAndLoopBack(localAddresses)
	if len(localAddresses) == 0 {
		return nil
	}
	announceMsg := &pb.DiscoveryMsg{
		Type: pb.DiscoveryMsg_Announce,
		PInfos: []*pb.PeerInfo{
			{
				Pid: p.h.ID().String(),
				Addr: catutil.SliceTransformType(localAddresses, func(_ int, item ma.Multiaddr) string {
					return util.PIDAndNetAddrToMultiAddr(p.h.ID(), item).String()
				}),
			},
		},
	}
	bz, err := proto.Marshal(announceMsg)
	if err != nil {
		p.logger.Debug().Msg("failed to marshal discovery msg.").Err(err).Done()
		return err
	}
	for _, rp := range allPeers {
		rPID := rp.PID
		safe.LoggerGo(p.logger, func() {
			err := p.h.SendMsg(protocolID, rPID, bz)
			if err != nil {
				p.logger.Warn().
					Msg("failed to send discovery msg.").
					Str("type", pb.DiscoveryMsg_Announce.String()).
					Err(err).Done()
			}
		})
	}
	return nil
}

func (p *ProtocolDiscovery) SearchPeers(ctx context.Context, serviceName string) (<-chan []ma.Multiaddr, error) {
	serviceName = strings.TrimSpace(serviceName)
	p.mu.RLock()
	_, ok := p.searchingM[serviceName]
	p.mu.RUnlock()
	if ok {
		return nil, ErrSearching
	}
	searchC := make(chan []ma.Multiaddr)
	p.mu.Lock()
	p.searchingM[serviceName] = searchC
	p.mu.Unlock()

	safe.LoggerGo(p.logger, func() {
		p.searchTask(ctx, serviceName)
	})

	return searchC, nil
}

// createProtocolIDWithServiceName creates a protocol ID based on the service name.
func (p *ProtocolDiscovery) createProtocolIDWithServiceName(serviceName string) protocol.ID {
	return protocol.ID(protocolDiscoveryPIDPrefix + serviceName)
}

// handleSearchReq handles the search request.
func (p *ProtocolDiscovery) handleSearchReq(senderPID peer.ID, msg *pb.DiscoveryMsg, protocol protocol.ID) {
	p.logger.Debug().
		Msg("search request.").
		Str("searcher", senderPID.String()).
		Str("protocol", protocol.String()).
		Any("infos", msg.PInfos).
		Done()
	var (
		knownPeers = types.NewSet[peer.ID]() // Create a set to store known peers.
		searchSize = int(msg.Size)           // Get the requested search size.
	)

	knownPeers.Put(senderPID) // Add the sender to the set of known peers.
	for _, info := range msg.PInfos {
		knownPeers.Put(peer.ID(info.Pid)) // Add each peer ID in the message to the set of known peers.
	}

	if searchSize > p.maxSearchSize {
		searchSize = p.maxSearchSize // Limit the search size to the maximum allowed size.
	}

	peersFound := p.h.PeerStore().PeersSupportingAllProtocols(protocol) // Find peers supporting the specified protocol.
	pInfoList := make([]*pb.PeerInfo, 0, len(peersFound))               // Create a list to store the peer information.
	for _, pid := range peersFound {
		if knownPeers.Exist(pid) {
			continue // Skip known peers.
		}
		netAddresses := p.h.PeerStore().GetAddresses(pid) // Get the network addresses of the peer.
		if len(netAddresses) == 0 {
			continue // Skip peers with no network addresses.
		}
		pInfoList = append(pInfoList, &pb.PeerInfo{
			Pid: pid.String(),
			Addr: catutil.SliceTransformType(netAddresses, func(_ int, item ma.Multiaddr) string {
				return item.String()
			}),
		})
		if len(pInfoList) >= searchSize {
			break // Stop adding peer information if the requested search size is reached.
		}
	}

	resMsg := &pb.DiscoveryMsg{
		Type:   pb.DiscoveryMsg_SearchRes,
		PInfos: pInfoList,
	}
	bz, err := proto.Marshal(resMsg) // Marshal the response message.
	if err != nil {
		p.logger.Error().Msg("failed to marshal discovery msg.").Err(err).Done()
		return
	}

	err = p.h.SendMsg(protocol, senderPID, bz) // Send the response message.
	if err != nil {
		p.logger.Debug().
			Msg("failed to send discovery search response msg.").
			Str("remote-pid", senderPID.String()).
			Err(err).
			Done()
		return
	}
}

// handleSearchRes handles the search response.
func (p *ProtocolDiscovery) handleSearchRes(serviceName string, msg *pb.DiscoveryMsg) {
	p.logger.Debug().Msg("search response.").Str("service", serviceName).Any("infos", msg.PInfos).Done()

	if len(msg.PInfos) == 0 {
		return // If no peer information is included in the message, return.
	}

	for _, pInfo := range msg.PInfos {
		pid := peer.ID(pInfo.Pid)
		if p.h.ConnectionManager().Connected(pid) || !p.h.ConnectionManager().Allowed(pid) || p.h.ID() == pid {
			continue // Skip connected peers, disallowed peers, and self.
		}

		addrs := make([]ma.Multiaddr, 0, len(pInfo.Addr))
		for _, s := range pInfo.Addr {
			netAddr, err := ma.NewMultiaddr(s) // Parse the network address.
			if err != nil {
				p.logger.Debug().Msg("failed to parse addr to multiaddr.").Err(err).Done()
				return
			}
			addrs = append(addrs, util.PIDAndNetAddrToMultiAddr(pid, netAddr)) // Convert the peer ID and network address to a multiaddress.
		}
		addrs = catutil.SliceUnionBy(func(_ int, item ma.Multiaddr) string {
			return item.String()
		}, addrs)

		p.mu.RLock()
		c, ok := p.searchingM[serviceName] // Get the channel associated with the service name.
		if !ok {
			p.mu.RUnlock()
			return // If the channel does not exist, return.
		}
		c <- addrs // Send the discovered addresses to the channel.
		p.mu.RUnlock()
	}
}

// createProtocolMsgHandler creates a message payload handler for the given service name and protocol ID.
func (p *ProtocolDiscovery) createProtocolMsgHandler(
	serviceName string,
	protocolID protocol.ID,
) handler.MsgPayloadHandler {
	return func(senderPID peer.ID, msgPayload []byte) {
		// unmarshal payload
		msg := &pb.DiscoveryMsg{}
		err := proto.Unmarshal(msgPayload, msg)
		if err != nil {
			p.logger.Error().Msg("failed to unmarshal discovery msg.").Err(err).Done()
			return
		}

		switch msg.Type {
		case pb.DiscoveryMsg_Announce:
			if len(msg.PInfos) == 0 {
				p.logger.Debug().
					Msg("empty peer infos.").Str("type", pb.DiscoveryMsg_Announce.String()).Done()
				return
			}
			pid := peer.ID(msg.PInfos[0].Pid)
			if p.h.PeerStore().GetFirstAddress(pid) != nil {
				if err := p.h.PeerStore().AddProtocol(pid, protocolID); err != nil {
					p.logger.Debug().
						Msg("failed to add protocol id.").
						Str("type", pb.DiscoveryMsg_Announce.String()).
						Err(err).
						Done()
					return
				}
			}
		case pb.DiscoveryMsg_SearchReq:
			p.handleSearchReq(senderPID, msg, protocolID)
		case pb.DiscoveryMsg_SearchRes:
			p.handleSearchRes(serviceName, msg)
		default:
			p.logger.Warn().
				Msg("unknown discovery msg type.").
				Str("type", msg.Type.String()).
				Done()
		}
	}
}

// sendSearchReqToOthers sends a search request message to other peers supporting the specified protocol.
func (p *ProtocolDiscovery) sendSearchReqToOthers(protocolID protocol.ID) {
	// Find peers supporting the specified protocol
	peersFound := p.h.PeerStore().PeersSupportingAllProtocols(protocolID)
	if len(peersFound) == 0 {
		return
	}

	p.logger.Debug().Msg("sending search request to others.").Any("others", peersFound).Done()

	// Transform peer IDs to PeerInfo struct
	pInfos := catutil.SliceTransformType(peersFound, func(_ int, item peer.ID) *pb.PeerInfo {
		return &pb.PeerInfo{
			Pid:  item.String(),
			Addr: nil,
		}
	})

	// Create search request message
	req := &pb.DiscoveryMsg{
		Type:   pb.DiscoveryMsg_SearchReq,
		PInfos: pInfos,
		Size:   uint32(p.maxSearchSize),
	}

	// Marshal the message into bytes
	bz, err := proto.Marshal(req)
	if err != nil {
		p.logger.Error().Msg("failed to marshal discovery msg.").Err(err).Done()
		return
	}

	// Shuffle the list of peers
	peersFound = catutil.SliceShuffle(peersFound)

	// Send search request to selected peers
	for i, pid := range peersFound {
		if p.asksForPeers > 0 && i == p.asksForPeers {
			break
		}
		err = p.h.SendMsg(protocolID, pid, bz)
		if err != nil {
			p.logger.Debug().
				Msg("failed to send search request msg.").
				Str("remote-pid", pid.String()).
				Err(err).
				Done()
			return
		}
	}
}

// searchTask is a task that performs the search operation for a specific service.
// It continuously sends search requests to other peers at a specified interval.
func (p *ProtocolDiscovery) searchTask(ctx context.Context, serviceName string) {
	defer func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Clean up the search task
		close(p.searchingM[serviceName])
		delete(p.searchingM, serviceName)
	}()

	// Create the protocol ID for the specified service name
	protocolID := p.createProtocolIDWithServiceName(serviceName)

	ticker := time.NewTicker(time.Second)
	if p.searchTimeout > 0 {
		for {
			select {
			case <-time.After(p.searchTimeout):
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticker.Stop()
				// Send search request to other peers
				p.sendSearchReqToOthers(protocolID)

				ticker.Reset(p.searchInterval)
			}
		}
	} else {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticker.Stop()
				// Send search request to other peers
				p.sendSearchReqToOthers(protocolID)

				ticker.Reset(p.searchInterval)
			}
		}
	}
}

// applyOptions applies the provided options to the ProtocolDiscovery object.
func (p *ProtocolDiscovery) applyOptions(opts ...ProtocolDiscoveryOption) {
	for _, opt := range opts {
		opt(p)
	}
}
