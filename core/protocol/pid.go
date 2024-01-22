package protocol

// ID is an identifier used to mark the module to which the network message belongs.
type ID string

const (
	// TestingProtocolID is a protocol ID for testing purposes.
	TestingProtocolID ID = "/_testing"
)

func (i ID) String() string {
	return string(i)
}

// ParseStringsToProtocolIDs converts a string slice to a ID slice.
func ParseStringsToProtocolIDs(strs []string) []ID {
	res := make([]ID, len(strs))
	for idx := range strs {
		res[idx] = ID(strs[idx])
	}
	return res
}

// ParseProtocolIDsToStrings converts a ID slice to a string slice.
func ParseProtocolIDsToStrings(pids []ID) []string {
	res := make([]string, len(pids))
	for i := range pids {
		res[i] = string(pids[i])
	}
	return res
}
