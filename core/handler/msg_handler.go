package handler

import "github.com/rambollwong/rainbowbee/core/peer"

// MsgPayloadHandler is a function type used to handle the message payload received from a sender.
type MsgPayloadHandler func(senderPID peer.ID, msgPayload []byte)

// SubMsgHandler is a function type used to handle the message payload received from the PubSub topic network.
type SubMsgHandler func(publisher peer.ID, topic string, msg []byte)
