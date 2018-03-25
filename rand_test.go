// +build go1.8

package plock

import "math/rand"

func randUint64() uint64 {
	return rand.Uint64()
}
