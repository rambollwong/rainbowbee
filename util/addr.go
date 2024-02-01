package util

import (
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/peer"
)

const pidAddrPrefix = "/p2p/"

// SplitAddrToTransportAndPID resolves a network address and peer.ID from the given multiaddress.
// For example, if the multiaddress is "/ip4/127.0.0.1/tcp/8080/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4",
// it returns "/ip4/127.0.0.1/tcp/8080" as the network address and "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" as the peer.ID.
// If the multiaddress is "/ip4/127.0.0.1/tcp/8080", it returns "/ip4/127.0.0.1/tcp/8080" as the network address and an empty string as the peer.ID.
func SplitAddrToTransportAndPID(addr ma.Multiaddr) (ma.Multiaddr, peer.ID) {
	if addr == nil {
		return nil, ""
	}
	transport, p2pPart := ma.SplitLast(addr)
	if p2pPart == nil || p2pPart.Protocol().Code != ma.P_P2P {
		return addr, ""
	}
	var pid peer.ID
	pidStr, err := p2pPart.ValueForProtocol(ma.P_P2P)
	if err == nil {
		pid = peer.ID(pidStr)
	}
	return transport, pid
}

// PIDToMultiAddr converts a peer.ID to a p2p multiaddress.
// For example, "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" becomes "/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4".
func PIDToMultiAddr(pid peer.ID) ma.Multiaddr {
	return ma.StringCast(pidAddrPrefix + pid.String())
}

// PIDAndNetAddrToMultiAddr joins the peer.ID's p2p multiaddress with the given network address.
// For example, "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" and "/ip4/127.0.0.1/tcp/8080"
// become "/ip4/127.0.0.1/tcp/8080/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4".
func PIDAndNetAddrToMultiAddr(pid peer.ID, netAddr ma.Multiaddr) ma.Multiaddr {
	return ma.Join(netAddr, PIDToMultiAddr(pid))
}

// ContainsDNS returns true if the given address contains the DNS protocol, otherwise returns false
func ContainsDNS(addr ma.Multiaddr) bool {
	if addr != nil {
		for _, protocol := range addr.Protocols() {
			switch protocol.Code {
			case ma.P_DNS, ma.P_DNS4, ma.P_DNS6:
				return true
			}
		}
	}
	return false
}
