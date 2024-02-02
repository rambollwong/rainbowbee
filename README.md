# rainbow-bee
A p2p network library for Go.


## Roadmap

- [ ] Core
  - [x] interface
  - [x] crypto
- [ ] Networks
  - [x] TCP-network
  - [ ] UDP-Quic-network
  - [ ] WebSocket-network
- [x] PeerStore
  - [x] protocol book
  - [x] address book
  - [x] peer store
- [x] Components
  - [x] connection supervisor
  - [x] level connection manager
  - [x] protocol manager
  - [x] protocol exchanger
  - [x] send stream pool
  - [x] send stream pool manager
  - [x] receive stream manager
  - [x] blacklist
- [x] Host(MVP)
- [ ] Service
  - [ ] discovery (protocol based) (in progress)
  - [ ] broadcast
    - [ ] pubsub
  - [ ] group multicast
  - [ ] consensus
    - [ ] Raft
  