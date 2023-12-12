package protocol

// ProtocolID is an identifier used to mark the module to which the network message belongs.
type ProtocolID string

const (
	// TestingProtocolID is a protocol ID for testing purposes.
	TestingProtocolID ProtocolID = "/_testing"
)

// ParseStringsToProtocolIDs converts a string slice to a ProtocolID slice.
func ParseStringsToProtocolIDs(strs []string) []ProtocolID {
	res := make([]ProtocolID, len(strs))
	for idx := range strs {
		res[idx] = ProtocolID(strs[idx])
	}
	return res
}

// ParseProtocolIDsToStrings converts a ProtocolID slice to a string slice.
func ParseProtocolIDsToStrings(pids []ProtocolID) []string {
	res := make([]string, len(pids))
	for i := range pids {
		res[i] = string(pids[i])
	}
	return res
}
