package groupmulticast

import (
	"errors"
	"strings"
	"sync"

	"github.com/rambollwong/rainbowbee/core/groupmulticast"
	"github.com/rambollwong/rainbowbee/core/host"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/safe"
	"github.com/rambollwong/rainbowcat/util"
	"github.com/rambollwong/rainbowlog"
)

var (
	_ groupmulticast.GroupMulticast = (*PeerGroupManager)(nil)

	ErrNoPeerInGroup = errors.New("there is no peer in group")
)

type PeerGroups map[string][]peer.ID

type PeerGroupManager struct {
	h host.Host

	mu sync.RWMutex
	pg PeerGroups

	logger *rainbowlog.Logger
}

// AddPeerToGroup adds a group of peers to the specified peer group.
func (p *PeerGroupManager) AddPeerToGroup(groupName string, peers ...peer.ID) {
	// Lock the mutex to ensure exclusive access to the peer group map
	p.mu.Lock()
	defer p.mu.Unlock()

	// Retrieve the existing peer group or create a new one if it doesn't exist
	pg, ok := p.pg[groupName]
	if !ok {
		pg = make([]peer.ID, 0, len(peers))
	}

	// Append the new peers to the peer group
	pg = append(pg, peers...)
	p.pg[groupName] = pg
}

// RemovePeerFromGroup removes specified peers from the given peer group.
func (p *PeerGroupManager) RemovePeerFromGroup(groupName string, peers ...peer.ID) {
	// Lock the mutex to ensure exclusive access to the peer group map
	p.mu.Lock()
	defer p.mu.Unlock()

	// Retrieve the existing peer group
	pg, ok := p.pg[groupName]
	if !ok {
		return
	}

	// Exclude the specified peers from the peer group
	pg = util.SliceExcludeAll(pg, peers...)
	p.pg[groupName] = pg
}

// GroupSize returns the number of peers in the specified group.
func (p *PeerGroupManager) GroupSize(groupName string) int {
	// Lock the mutex for reading the peer group map
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Retrieve the peer group size for the given group name
	return len(p.pg[groupName])
}

// InGroup checks if the specified peer is a member of the given group.
func (p *PeerGroupManager) InGroup(groupName string, peer peer.ID) bool {
	// Lock the mutex for reading the peer group map
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Retrieve the peer group for the given group name
	pg, ok := p.pg[groupName]
	if !ok {
		return false
	}

	// Check if the specified peer is present in the peer group
	return util.SliceContains(pg, peer)
}

// RemoveGroup removes the specified group from the peer group map.
func (p *PeerGroupManager) RemoveGroup(groupName string) {
	// Lock the mutex to ensure exclusive access to the peer group map
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove the specified group from the peer group map
	delete(p.pg, groupName)
}

// SendToGroupSync sends a data message synchronously to all peers in the specified group.
func (p *PeerGroupManager) SendToGroupSync(groupName string, protocolID protocol.ID, data []byte) error {
	// Retrieve all peers in the group
	pg := p.getGroupPeers(groupName)

	// If there are no peers in the group, return an error
	if len(pg) == 0 {
		return ErrNoPeerInGroup
	}

	// Create a channel to receive errors
	errorC := make(chan error, len(pg))

	// Send the data message to all peers in the group
	var wg sync.WaitGroup
	for _, pid := range pg {
		if !p.h.ConnectionManager().Connected(pid) {
			// Skip peers that are not connected
			continue
		}
		wg.Add(1)

		// Send the message to the peer in a separate goroutine
		safe.LoggerGo(p.logger, func() {
			defer wg.Done()
			err := p.h.SendMsg(protocolID, pid, data)
			if err != nil {
				p.logger.Error().
					Msg("failed to send msg to peer.").
					Str("group", groupName).
					Str("protocol", protocolID.String()).Str("pid", pid.String()).
					Err(err).
					Done()
				errorC <- err
			}
		})
	}

	wg.Wait()
	close(errorC)

	if len(errorC) > 0 {
		errorArr := make([]string, 0, len(errorC))
		for err := range errorC {
			errorArr = append(errorArr, err.Error())
		}
		return errors.New("something error [" + strings.Join(errorArr, ",") + "]")
	}

	return nil
}

// SendToGroupAsync sends a data message asynchronously to all peers in the specified group.
func (p *PeerGroupManager) SendToGroupAsync(groupName string, protocolID protocol.ID, data []byte) <-chan error {
	// Retrieve all peers in the group
	pg := p.getGroupPeers(groupName)

	// Create a channel to receive errors
	errorC := make(chan error, len(pg))

	// If there are no peers in the group, return an error
	if len(pg) == 0 {
		errorC <- ErrNoPeerInGroup
		return errorC
	}

	// Send the data message to all peers in the group
	for _, pid := range pg {
		if !p.h.ConnectionManager().Connected(pid) {
			// Skip peers that are not connected
			continue
		}
		// Send the message to the peer in a separate goroutine
		safe.LoggerGo(p.logger, func() {
			err := p.h.SendMsg(protocolID, pid, data)
			if err != nil {
				p.logger.Error().
					Msg("failed to send msg to peer.").
					Str("group", groupName).
					Str("protocol", protocolID.String()).Str("pid", pid.String()).
					Err(err).
					Done()
				errorC <- err
			}
		})
	}

	return errorC
}

// getGroupPeers returns all the peers in the group whose name is the given group name.
func (p *PeerGroupManager) getGroupPeers(groupName string) []peer.ID {
	// Lock the mutex for reading the peer group map
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Retrieve the peer group for the given group name
	pg, ok := p.pg[groupName]
	if !ok {
		return nil
	}

	// Create a new slice and copy the peer IDs to it
	res := make([]peer.ID, len(pg))
	copy(res, pg)
	return res
}
