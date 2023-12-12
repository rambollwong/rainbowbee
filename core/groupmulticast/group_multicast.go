package groupmulticast

import (
	"rambollwong/rainbowbee/core/peer"
	"rambollwong/rainbowbee/core/protocol"
)

// GroupMulticast sends messages to all peers in a group.
type GroupMulticast interface {
	// AddPeerToGroup adds peers to the group with the given groupName.
	// If the group does not exist, it will be created.
	AddPeerToGroup(groupName string, peers ...peer.ID)

	// RemovePeerFromGroup removes peers from the group with the given groupName.
	RemovePeerFromGroup(groupName string, peers ...peer.ID)

	// GroupSize returns the count of peers in the group with the given groupName.
	// If the group does not exist, it returns 0.
	GroupSize(groupName string) int

	// InGroup returns whether the peer is in the group with the given groupName.
	// If the group does not exist, it returns false.
	InGroup(groupName string, peer peer.ID) bool

	// RemoveGroup removes the group with the given groupName.
	RemoveGroup(groupName string)

	// SendToGroupSync sends data synchronously to the peers in the group.
	// It waits until all data is successfully sent.
	SendToGroupSync(groupName string, protocolID protocol.ProtocolID, data []byte) error

	// SendToGroupAsync sends data asynchronously to the group without waiting.
	SendToGroupAsync(groupName string, protocolID protocol.ProtocolID, data []byte)
}
