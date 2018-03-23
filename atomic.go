package plock

import "sync/atomic"

//go:nosplit
func subUint64(addr *uint64, new uint64) uint64 {
	new = (^new) + 1 // add 2's complement
	return atomic.AddUint64(addr, new)
}

//go:nosplit
func subUint32(addr *uint32, new uint32) uint32 {
	new = (^new) + 1 // add 2's complement
	return atomic.AddUint32(addr, new)
}

//go:nosplit
func xadd32(addr *uint32, new uint32) uint32 {
	x := atomic.AddUint32(addr, new)

	return x - new
}

//go:nosplit
func xadd64(addr *uint64, new uint64) uint64 {
	x := atomic.AddUint64(addr, new)

	return x - new
}
