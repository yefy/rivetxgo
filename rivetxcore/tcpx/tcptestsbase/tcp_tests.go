package tcptestsbase

import (
	"math"
)

const TestDataQuit = int32(math.MaxInt32)
const TestDataLen = 4

func Int32To4Bytes(num int32) [4]byte {
	return [4]byte{
		byte(num >> 24),
		byte(num >> 16),
		byte(num >> 8),
		byte(num),
	}
}

func Bytes4ToInt32(data [4]byte) int32 {
	return int32(data[0])<<24 |
		int32(data[1])<<16 |
		int32(data[2])<<8 |
		int32(data[3])
}
