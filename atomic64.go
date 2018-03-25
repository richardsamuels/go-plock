// build
package plock

import "sync/atomic"

// subUint64 atomically subtracts new from the uint64 at addr. This relies
// on the XADD instruction
//go:nosplit
func subUint64(addr *uint64, new uint64) uint64 {
	new = (^new) + 1 // add 2's complement
	return atomic.AddUint64(addr, new)
}

// xadd64 atomically adds new to the uint64 at addr, and returns the old value
// at addr
//go:nosplit
func xadd64(addr *uint64, new uint64) uint64 {
	x := atomic.AddUint64(addr, new)

	return x - new
}
