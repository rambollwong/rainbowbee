package components

import (
	"errors"
	"sync"

	"github.com/rambollwong/rainbowbee/core/handler"
	"github.com/rambollwong/rainbowbee/core/manager"
	"github.com/rambollwong/rainbowbee/core/peer"
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowbee/core/store"
	"github.com/rambollwong/rainbowcat/util"
)

var (
	ErrProtocolIDRegistered    = errors.New("protocol id has registered")
	ErrProtocolIDNotRegistered = errors.New("protocol id is not registered")

	_ manager.ProtocolManager = (*ProtocolManager)(nil)
)

// ProtocolManager manages protocols and protocol message handlers for peers.
type ProtocolManager struct {
	mu                  sync.RWMutex
	handlers            map[protocol.ID]handler.MsgPayloadHandler
	protocolBook        store.ProtocolBook
	supportedCallback   manager.ProtocolSupportNotifyFunc
	unsupportedCallback manager.ProtocolSupportNotifyFunc
}

// NewProtocolManager creates a new instance of ProtocolManager with the provided ProtocolBook.
// It initializes the internal fields of the ProtocolManager
// and returns a value that implements the manager.ProtocolManager interface.
func NewProtocolManager(protocolBook store.ProtocolBook) manager.ProtocolManager {
	return &ProtocolManager{
		mu:                  sync.RWMutex{},
		handlers:            make(map[protocol.ID]handler.MsgPayloadHandler),
		protocolBook:        protocolBook,
		supportedCallback:   nil,
		unsupportedCallback: nil,
	}
}

// RegisterMsgPayloadHandler registers a protocol and associates a handler.MsgPayloadHandler with it.
// It returns an error if the registration fails.
func (p *ProtocolManager) RegisterMsgPayloadHandler(protocolID protocol.ID, handler handler.MsgPayloadHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.handlers[protocolID]
	if ok {
		return ErrProtocolIDRegistered
	}
	p.handlers[protocolID] = handler
	return nil
}

// UnregisterMsgPayloadHandler unregisters a previously registered protocol.
// It returns an error if the un-registration fails.
func (p *ProtocolManager) UnregisterMsgPayloadHandler(protocolID protocol.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.handlers[protocolID]
	if !ok {
		return ErrProtocolIDNotRegistered
	}
	delete(p.handlers, protocolID)
	return nil
}

// Registered checks if a protocol is registered and supported.
// It returns true if the protocol is supported, and false otherwise.
func (p *ProtocolManager) Registered(protocolID protocol.ID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.handlers[protocolID]
	return ok
}

// Handler returns the message payload handler associated with the registered protocol.
// If the protocol is not registered or supported, it returns nil.
func (p *ProtocolManager) Handler(protocolID protocol.ID) handler.MsgPayloadHandler {
	p.mu.RLock()
	defer p.mu.RUnlock()
	h, _ := p.handlers[protocolID]
	return h
}

// SupportedByPeer checks if a peer with the given peer.ID supports a specific protocol.
// It returns true if the protocol is supported by the peer, and false otherwise.
// If the peer is not connected, it returns false.
func (p *ProtocolManager) SupportedByPeer(pid peer.ID, protocolID protocol.ID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.protocolBook.ContainsProtocol(pid, protocolID)
}

// SupportedProtocolsOfPeer returns a list of protocol.IDs that are supported by the peer with the given peer.ID.
// If the peer is not connected or doesn't support any protocols, it returns an empty list.
func (p *ProtocolManager) SupportedProtocolsOfPeer(pid peer.ID) []protocol.ID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.protocolBook.GetProtocols(pid)
}

// notifyIfChanged notifies the registered callback functions if there are changes in the supported protocols of a peer.
// It is called internally by SetSupportedProtocolsOfPeer method.
func (p *ProtocolManager) notifyIfChanged(pid peer.ID, protocols []protocol.ID) {
	if len(protocols) == 0 {
		return
	}

	if p.supportedCallback != nil {
		for _, ID := range protocols {
			if !p.protocolBook.ContainsProtocol(pid, ID) {
				p.supportedCallback(ID, pid)
			}
		}
	}

	if p.unsupportedCallback != nil {
		supported := p.protocolBook.GetProtocols(pid)
		unsupported := util.SliceExcludeAll(supported, protocols...)
		for _, ID := range unsupported {
			p.unsupportedCallback(ID, pid)
		}
	}
}

// SetSupportedProtocolsOfPeer stores the protocols supported by the peer with the given peer.ID.
// It notifies the registered callback functions if there are changes in the supported protocols of the peer.
// It returns an error if the protocol storage fails.
func (p *ProtocolManager) SetSupportedProtocolsOfPeer(pid peer.ID, protocolIDs []protocol.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.notifyIfChanged(pid, protocolIDs)
	return p.protocolBook.SetProtocols(pid, protocolIDs)
}

// CleanPeer removes all records of protocols supported by the peer with the given peer.ID.
// It notifies the registered callback functions for any unsupported protocols.
// It returns an error if the cleanup fails.
func (p *ProtocolManager) CleanPeer(pid peer.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	supported := p.protocolBook.GetProtocols(pid)
	for _, ID := range supported {
		p.unsupportedCallback(ID, pid)
	}
	return p.protocolBook.ClearProtocol(pid)
}

// SetProtocolSupportedNotifyFunc sets the callback function to be invoked when a protocol is supported by a peer.
// It returns an error if the setting fails.
func (p *ProtocolManager) SetProtocolSupportedNotifyFunc(notifyFunc manager.ProtocolSupportNotifyFunc) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.supportedCallback = notifyFunc
	return nil
}

// SetProtocolUnsupportedNotifyFunc sets the callback function to be invoked when a protocol is unsupported by a peer.
// It returns an error if the setting fails.
func (p *ProtocolManager) SetProtocolUnsupportedNotifyFunc(notifyFunc manager.ProtocolSupportNotifyFunc) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.unsupportedCallback = notifyFunc
	return nil
}
