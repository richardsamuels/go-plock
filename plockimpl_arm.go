package plock

import (
	"fmt"
	"runtime"

	"sync/atomic"
)

// PMutex is an implementation of Willy Tarreau's Progressive locks (full post
// at: http://wtarreau.blogspot.com/2018/02/progressive-locks-fast-upgradable.html)

// Progressive locks offer the conventional Read and Write locks of sync.RWMutex,
// but offer 2 additional states: Atomic Write Lock, allowing for multiple writers,
// that have the obligation to interact with protected data atomically; and
// Seek Lock, which is an exclusive reader that can quickly upgrade its lock
// to Write
type PMutex struct {
	lock uint32
}

const (
	leftShiftVal  = 16
	rightShiftVal = 18
)

// String returns a string representing the internal lock state
// The string built by this function reflects a single moment in time,
// _NOT_ the current moment; i.e. Do NOT rely on this for anything other
// than debugging
func (p *PMutex) String() string {
	v := atomic.LoadUint32(&p.lock)
	addr := fmt.Sprintf("Addr: %d;", &p.lock)
	if v == 0 {
		return addr + " U"
	}

	hasWriter := v&plock32WLAny != 0
	hasSeeker := v&plock32SLAny != 0
	hasReader := v&plock32RLAny != 0

	numReaders := (v << leftShiftVal) >> rightShiftVal
	if hasWriter && !hasSeeker && !hasReader {
		numWriters := v >> rightShiftVal
		return fmt.Sprintf("%s A; writers: %d", addr, numWriters)
	}

	if hasReader && !hasSeeker && !hasWriter {
		if numReaders == 1 {
			return addr + " R; readers: self only"
		}

		return fmt.Sprintf("%s R; readers: %d", addr, numReaders-1)
	}

	s := addr + " "
	if hasReader {
		s += "R"
	}
	if hasSeeker {
		s += "+S"
	}
	if hasWriter {
		if numReaders == 1 {
			s += "+W; readers: self only"
		} else {
			s += fmt.Sprintf("+W; waiting for readers: %d", numReaders-1)

		}
	}

	return s
}

//go:nosplit
func (p *PMutex) tryRLock() bool {
	const setR = plock32RL1
	const maskR = plock32WLAny

	// Since all writes to this value are atomic, load is unnecessary,
	// but it makes the race detector happy
	if (atomic.LoadUint32(&p.lock) & maskR) != 0 {
		return false
	}
	if xadd32(&p.lock, setR)&maskR == 0 {
		return true
	}
	_ = subUint32(&p.lock, setR)

	return false
}

// RLock acquires a Read lock. This method will block until the lock is acquired,
// yielding the goroutine to the scheduler after every failure to acquire
func (p *PMutex) RLock() {
	for {
		if p.tryRLock() {
			break
		}

		runtime.Gosched()
	}
}

// RUnlock releases an existing Read Lock
func (p *PMutex) RUnlock() {
	const val = plock32RL1
	_ = subUint32(&p.lock, val)
}

func (p *PMutex) tryRToA() bool {
	plr := atomic.LoadUint32(&p.lock) & plock32SLAny
	if plr == 0 {
		plr = xadd32(&p.lock, plock32WL1-plock32RL1)
		for {
			if plr&plock32SLAny != 0 {
				_ = subUint32(&p.lock, plock32WL1-plock32RL1)
				break
			}

			plr &= plock32RLAny
			if plr != 0 {
				break
			}
			plr = atomic.LoadUint32(&p.lock)
		}
	}

	return plr != 0
}

// RToA upgrades an existing Read Lock to an Atomic Write Lock
func (p *PMutex) RToA() {
	for {
		if p.tryRToA() {
			break
		}

		runtime.Gosched()
	}
}

func (p *PMutex) tryRToW() bool {
	const setR = plock32WL1 | plock32SL1
	const maskR = plock32WLAny | plock32SLAny
	var plr uint32
	for {
		plr = xadd32(&p.lock, setR)
		if plr&maskR != 0 {
			if xadd32(&p.lock, (^setR)+1) != 0 {
				break
			}
			continue
		}

		for plr != 0 {
			plr = atomic.LoadUint32(&p.lock) - (plock32WL1 | plock32SL1 | plock32RL1) //nolint:megacheck
			break
		}
	}

	return plr == 0
}

// RToW upgrades an existing Read Lock to a Write Lock.
func (p *PMutex) RToW() {
	for {
		if p.tryRToW() {
			break
		}

		runtime.Gosched()
	}
}

func (p *PMutex) tryRToS() bool {
	plr := atomic.LoadUint32(&p.lock)
	if plr&(plock32WLAny|plock32SLAny) == 0 {
		plr = xadd32(&p.lock, plock32SL1) & (plock32WLAny | plock32SLAny)
		if plr != 0 {
			_ = subUint32(&p.lock, plock32SL1)
		}

	}

	return plr == 0
}

// RToS upgrades an existing Read Lock to a Seek Lock
func (p *PMutex) RToS() {
	for {
		if p.tryRToS() {
			break
		}

		runtime.Gosched()
	}
}

//go:nosplit
func (p *PMutex) tryWLock() bool {
	const setR = plock32WL1 | plock32SL1 | plock32RL1
	const maskR = plock32WLAny | plock32SLAny

	if xadd32(&p.lock, setR)&maskR == 0 {
		return true
	}
	_ = subUint32(&p.lock, setR)

	return false
}

// WLock acquires a Write Lock, blocking until all current readers unlock
func (p *PMutex) WLock() {
	const setR = plock32WL1 | plock32SL1 | plock32RL1

	// acquire lock
	for {
		if p.tryWLock() {
			break
		}
		runtime.Gosched()
	}

	// wait for readers to leave
	for {
		if (atomic.LoadUint32(&p.lock) - setR) == 0 {
			break
		}
		// yield here in the this half acquired state;
		// this allows readers the opportunity to finish up, and prevents
		// new readers/writers from entering
		runtime.Gosched()
	}
}

// WUnlock releases an existing Write Lock.
func (p *PMutex) WUnlock() {
	const val = plock32WL1 | plock32SL1 | plock32RL1
	_ = subUint32(&p.lock, val)
}

// WToR downgrades an existing Write Lock to a Read Lock
func (p *PMutex) WToR() {
	const val = plock32WL1 | plock32SL1
	_ = subUint32(&p.lock, val)
}

// WToR downgrades an existing Write Lock to a Seek Lock
func (p *PMutex) WToS() {
	const val = plock32WL1
	_ = subUint32(&p.lock, val)
}

//go:nosplit
func (p *PMutex) trySLock() bool {
	const setR = plock32SL1 | plock32RL1
	const maskR = plock32WLAny | plock32SLAny
	if atomic.LoadUint32(&p.lock)&maskR == 0 {
		if xadd32(&p.lock, setR) == 0 {
			return true
		}
		_ = subUint32(&p.lock, setR)
	}
	return false
}

// SLock acquires a Seek Lock. This state allows for an exclusive reader,
// which has the ability to quickly upgrade to a Write Lock if needed
func (p *PMutex) SLock() {
	for {
		if p.trySLock() {
			break
		}
		runtime.Gosched()
	}
}

// SUnlock releases an existing Seek Lock
func (p *PMutex) SUnlock() {
	const val = plock32SL1 + plock32RL1
	_ = subUint32(&p.lock, val)
}

// SToR downgrades an existing Seek Lock to a Read Lock
func (p *PMutex) SToR() {
	const val = plock32SL1
	_ = subUint32(&p.lock, val)
}

// SToR upgrades an existing Seek Lock to a Write Lock
func (p *PMutex) SToW() {
	t := xadd32(&p.lock, plock32WL1)
	for {
		if t&plock32RLAny != plock32RL1 {
			t = atomic.LoadUint32(&p.lock)
		}
	}
}

//go:nosplit
func (p *PMutex) tryALock() bool {
	const setR = plock32WL1
	const maskR = plock32SLAny

	plr := atomic.LoadUint32(&p.lock) & maskR
	if plr == 0 {
		plr = xadd32(&p.lock, setR)
		for {
			if plr&maskR != 0 {
				_ = subUint32(&p.lock, setR)
				break
			}
			plr &= plock32RLAny
			if plr == 0 {
				break
			}
			plr = atomic.LoadUint32(&p.lock)
		}
	}

	return plr == 0
}

// ALock acquires an Atomic Write Lock. Atomic Write allows for multiple writers,
// however all writers must access the shared data atomically (ex: sync/atomic.*)
func (p *PMutex) ALock() {
	for {
		if p.tryALock() {
			break
		}
		runtime.Gosched()
	}
}

// AUnlock releases an Atomic Write Lock
func (p *PMutex) AUnlock() {
	const val = plock32WL1
	_ = subUint32(&p.lock, val)
}
