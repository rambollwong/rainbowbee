package peerstore

import (
	"sync"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/store"
)

var _ store.AddressBook = (*addressBook)(nil)

type addressBook struct {
	book sync.Map
}

// AddAddress adds one or more Multiaddr to the address book for a peer ID.
func (a *addressBook) AddAddress(pid peer.ID, addr ...ma.Multiaddr) error {
	list, ok := a.book.Load(pid)
	if !ok {
		l := newAddrList()
		list, ok = a.book.LoadOrStore(pid, l)
		if !ok {
			list = l
		}
	}
	for _, ma := range addr {
		newAddr := ma
		list.(*addrList).Save(newAddr)
	}
	return nil
}

// SetAddresses sets the Multiaddr for a peer ID, replacing any existing addresses.
func (a *addressBook) SetAddresses(pid peer.ID, addresses []ma.Multiaddr) error {
	list := newAddrList()
	list.Append(addresses...)
	a.book.Store(pid, list)
	return nil
}

// RemoveAddress removes one or more Multiaddr from the address book for a peer ID.
func (a *addressBook) RemoveAddress(pid peer.ID, addr ...ma.Multiaddr) error {
	list, ok := a.book.Load(pid)
	if ok {
		list.(*addrList).Remove(addr...)
	}
	return nil
}

// GetFirstAddress returns the first Multiaddr from the address book for a peer ID.
func (a *addressBook) GetFirstAddress(pid peer.ID) ma.Multiaddr {
	var res ma.Multiaddr
	list, ok := a.book.Load(pid)
	if ok {
		arr := list.(*addrList).List()
		if len(arr) > 0 {
			res = arr[0]
		}
	}
	return res
}

// GetAddresses returns all the Multiaddr from the address book for a peer ID.
func (a *addressBook) GetAddresses(pid peer.ID) []ma.Multiaddr {
	list, ok := a.book.Load(pid)
	if ok {
		return list.(*addrList).List()
	}
	return nil
}

type addrList struct {
	mu sync.RWMutex
	l  []ma.Multiaddr
}

// newAddrList creates a new instance of addrList.
func newAddrList() *addrList {
	return &addrList{
		mu: sync.RWMutex{},
		l:  make([]ma.Multiaddr, 0),
	}
}

// Append appends one or more Multiaddr to the list.
func (s *addrList) Append(addr ...ma.Multiaddr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ma := range addr {
		ma := ma
		s.l = append(s.l, ma)
	}
}

// Save appends unique Multiaddr to the list.
func (s *addrList) Save(addr ...ma.Multiaddr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tempM := make(map[string]struct{})
	for _, ma := range s.l {
		tempM[ma.String()] = struct{}{}
	}
	for _, ma := range addr {
		ma := ma
		if _, ok := tempM[ma.String()]; !ok {
			s.l = append(s.l, ma)
		}
	}
}

// Remove removes one or more Multiaddr from the list.
func (s *addrList) Remove(addr ...ma.Multiaddr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.l) == 0 {
		return
	}
	res := make([]ma.Multiaddr, 0)
	m := make(map[string]struct{})
	for _, ma := range addr {
		m[ma.String()] = struct{}{}
	}
	for _, ma := range s.l {
		ma := ma
		tmp := ma.String()
		if _, ok := m[tmp]; !ok {
			res = append(res, ma)
		}
	}
	s.l = res
}

// List returns a copy of the Multiaddr list.
func (s *addrList) List() []ma.Multiaddr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ma.Multiaddr, len(s.l))
	copy(res, s.l)
	return res
}

// Size returns the number of Multiaddr in the list.
func (s *addrList) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.l)
}
