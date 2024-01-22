package util

import (
	"github.com/rambollwong/rainbowbee/core/network"
	"github.com/rambollwong/rainbowcat/util"
)

const (
	defaultBatchSize = 4 << 10 // 4KB
)

// ReadPackageLength reads the package length from the receive stream.
// It expects the length to be encoded as 8 bytes in big-endian format.
// Returns the package length, the length bytes, and any error encountered.
func ReadPackageLength(stream network.ReceiveStream) (uint64, []byte, error) {
	lengthBytes, err := ReadPackageData(stream, 8)
	if err != nil {
		return 0, nil, err
	}
	length := util.BytesToUint64(lengthBytes)
	return length, lengthBytes, nil
}

// ReadPackageData reads the specified length of data from the receive stream.
// It reads the data in batches of defaultBatchSize (4KB) to optimize performance.
// Returns the read data and any error encountered.
func ReadPackageData(stream network.ReceiveStream, length uint64) ([]byte, error) {
	var readLength uint64
	var batchSize uint64 = defaultBatchSize
	result := make([]byte, length)
	for length > 0 {
		if length < batchSize {
			batchSize = length
		}
		bytes := make([]byte, batchSize)
		c, err := stream.Read(bytes)
		if err != nil {
			return nil, err
		}
		length = length - uint64(c)
		copy(result[readLength:], bytes[0:c])
		readLength = readLength + uint64(c)
	}
	return result, nil
}
