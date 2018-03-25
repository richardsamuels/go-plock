package plock

import "sync/atomic"

// subUint32 atomically subtracts new from the uint32 at addr. This relies
// on the XADD instruction
//go:nosplit
func subUint32(addr *uint32, new uint32) uint32 {
	new = (^new) + 1 // add 2's complement
	return atomic.AddUint32(addr, new)
}

// xadd32 atomically adds new to the uint32 at addr, and returns the old value
// at addr
//go:nosplit
func xadd32(addr *uint32, new uint32) uint32 {
	x := atomic.AddUint32(addr, new)

	return x - new
}
