package components

import (
	"net"
	"strings"

	"github.com/rambollwong/rainbowbee/core/blacklist"
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowcat/types"
)

var (
	_ blacklist.PeerBlackList = (*BlackList)(nil)
)

// BlackList is a component that implements the PeerBlackList interface to manage a blacklist of network addresses or peer IDs.
type BlackList struct {
	ipAndPort *types.Set[string]  // Set of IP:Port strings
	pidSet    *types.Set[peer.ID] // Set of peer IDs
}

// NewBlacklist creates a new instance of the BlackList component.
func NewBlacklist() *BlackList {
	return &BlackList{
		ipAndPort: types.NewSet[string](),
		pidSet:    types.NewSet[peer.ID](),
	}
}

// AddPeerID adds a peer ID to the blacklist.
func (b *BlackList) AddPeerID(pid peer.ID) {
	b.pidSet.Put(pid)
}

// RemovePeerID removes a peer ID from the blacklist.
// If the peer ID does not exist in the blacklist, it is a no-op.
func (b *BlackList) RemovePeerID(pid peer.ID) {
	b.pidSet.Remove(pid)
}

// AddIPAndPort adds a network address or IP to the blacklist.
// The address should be in the format "IP:Port".
func (b *BlackList) AddIPAndPort(ipAndPort string) {
	ok, n := b.checkIPAndPort(ipAndPort)
	if ok {
		b.ipAndPort.Put(n)
	}
}

// RemoveIPAndPort removes a network address or IP from the blacklist.
// If the address does not exist in the blacklist, it is a no-op.
func (b *BlackList) RemoveIPAndPort(ipAndPort string) {
	ok, n := b.checkIPAndPort(ipAndPort)
	if ok {
		b.ipAndPort.Remove(n)
	}
}

// BlackConn checks if the remote peer ID or network address of the given connection is blacklisted.
// Returns true if the connection is blacklisted, otherwise false.
func (b *BlackList) BlackConn(conn network.Connection) bool {
	if conn == nil {
		return false
	}
	remotePid := conn.RemotePeerID()
	if b.pidSet.Exist(remotePid) {
		return true
	}
	netAddrStr := conn.RemoteNetAddr().String()
	if b.ipAndPort.Exist(netAddrStr) {
		return true
	}
	ip6 := strings.Contains(netAddrStr, "[")
	ip, _, err := net.SplitHostPort(netAddrStr)
	if err != nil {
		return false
	}
	if ip6 {
		ip = "[" + ip + "]"
	}
	if b.ipAndPort.Exist(ip) {
		return true
	}
	return false
}

// BlackPID checks if a given peer ID is blacklisted.
// Returns true if the peer ID is blacklisted, otherwise false.
func (b *BlackList) BlackPID(pid peer.ID) bool {
	return b.pidSet.Exist(pid)
}

// checkIPAndPort checks if the given IP:Port string is valid.
// Returns true and the validated IP:Port if it is valid, otherwise false and an empty string.
func (b *BlackList) checkIPAndPort(ipAndPort string) (bool, string) {
	_, _, err := net.SplitHostPort(ipAndPort)
	if err == nil {
		return true, ipAndPort
	}
	// not IP:Port format
	// Check if it is only an IP
	_, _, err = net.SplitHostPort(ipAndPort + ":80")
	if err == nil {
		return true, ipAndPort
	}
	// Maybe an IPv6 address?
	newIp := "[" + ipAndPort + "]"
	_, _, err = net.SplitHostPort(newIp + ":80")
	if err == nil {
		return true, newIp
	}
	return false, ""
}
