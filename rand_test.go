// +build go1.8

package plock

import "math/rand"

const randomType = "math/rand.Uint64"

func randUint64() uint64 {
	return rand.Uint64()
}
