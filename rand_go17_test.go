// +build !go1.8

package plock

import "math/rand"

const randomType = "uint32 bitshift"

func randUint64() uint64 {
	return (uint64(rand.Uint32()) << 32) + uint64(rand.Uint32())
}
