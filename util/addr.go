package util

import (
	"net"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/peer"
)

const pidAddrPrefix = "/p2p/"

// GetNetAddrAndPIDFromNormalMultiAddr resolves a network address and peer.ID from the given multiaddress.
// For example, if the multiaddress is "/ip4/127.0.0.1/tcp/8080/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4",
// it returns "/ip4/127.0.0.1/tcp/8080" as the network address and "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" as the peer.ID.
// If the multiaddress is "/ip4/127.0.0.1/tcp/8080", it returns "/ip4/127.0.0.1/tcp/8080" as the network address and an empty string as the peer.ID.
func GetNetAddrAndPIDFromNormalMultiAddr(addr ma.Multiaddr) (ma.Multiaddr, peer.ID) {
	var addrMa, pidMa ma.Multiaddr
	var pid peer.ID
	addrMa, pidMa = ma.SplitFunc(addr, func(component ma.Component) bool {
		return component.Protocol().Code == ma.P_P2P
	})
	if pidMa == nil {
		return addrMa, pid
	}
	pidStr, err := pidMa.ValueForProtocol(ma.P_P2P)
	if err == nil {
		pid = peer.ID(pidStr)
	}
	return addrMa, pid
}

// GetLocalAddresses returns a list of net.Addr that can be used and are bound to each network interface.
func GetLocalAddresses() ([]net.Addr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var res []net.Addr
	for _, i := range interfaces {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		res = append(res, addresses...)
	}
	return res, nil
}

// CreateMultiAddrWithPID converts a peer.ID to a p2p multiaddress.
// For example, "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" becomes "/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4".
func CreateMultiAddrWithPID(pid peer.ID) ma.Multiaddr {
	return ma.StringCast(pidAddrPrefix + pid.String())
}

// CreateMultiAddrWithPIDAndNetAddr joins the peer.ID's p2p multiaddress with the given network address.
// For example, "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4" and "/ip4/127.0.0.1/tcp/8080"
// become "/ip4/127.0.0.1/tcp/8080/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4".
func CreateMultiAddrWithPIDAndNetAddr(pid peer.ID, netAddr ma.Multiaddr) ma.Multiaddr {
	return ma.Join(netAddr, CreateMultiAddrWithPID(pid))
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
