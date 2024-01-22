package util

import (
	"github.com/rambollwong/rainbowbee/core/protocol"
	"github.com/rambollwong/rainbowcat/util"
)

// ConvertStrArrToProtocolIDs converts an array of strings to an array of protocol.IDs.
func ConvertStrArrToProtocolIDs(strArr ...string) []protocol.ID {
	return util.SliceTransformType(strArr, func(_ int, item string) protocol.ID {
		return protocol.ID(item)
	})
}

// ConvertProtocolIDsToStrArr converts an array of protocol.IDs to an array of strings.
func ConvertProtocolIDsToStrArr(protocols ...protocol.ID) []string {
	return util.SliceTransformType(protocols, func(_ int, item protocol.ID) string {
		return item.String()
	})
}
