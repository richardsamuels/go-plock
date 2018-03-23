package plock

import (
	"fmt"
	"runtime"
	"sync"

	"sync/atomic"
)

type PMutex struct {
	lock uint64
}

const (
	PLOCK64_RL_1   uint64 = 0x0000000000000004
	PLOCK64_RL_ANY uint64 = 0x00000000FFFFFFFC
	PLOCK64_SL_1   uint64 = 0x0000000100000000
	PLOCK64_SL_ANY uint64 = 0x0000000300000000
	PLOCK64_WL_1   uint64 = 0x0000000400000000
	PLOCK64_WL_ANY uint64 = 0xFFFFFFFC00000000
)

const (
	PLOCK32_RL_1   uint32 = 0x00000004
	PLOCK32_RL_ANY uint32 = 0x0000FFFC
	PLOCK32_SL_1   uint32 = 0x00010000
	PLOCK32_SL_ANY uint32 = 0x00030000
	PLOCK32_WL_1   uint32 = 0x00040000
	PLOCK32_WL_ANY uint32 = 0xFFFC0000
)

// Locker interface just uses the W lock
func (p *PMutex) Unlock() {
	p.WUnlock()
}

func (p *PMutex) Lock() {
	p.WLock()
}

// String returns a string representing the internal lock state
// The string built by this function reflects a single moment in time,
// _NOT_ the current moment; i.e. Do NOT rely on this for anything other
// than debugging
func (p *PMutex) String() string {
	v := atomic.LoadUint64(&p.lock)
	addr := fmt.Sprintf("Addr: %d;", &p.lock)
	if v == 0 {
		return addr + " U"
	}

	hasWriter := v&PLOCK64_WL_ANY != 0
	hasSeeker := v&PLOCK64_SL_ANY != 0
	hasReader := v&PLOCK64_RL_ANY != 0

	numReaders := (v << 32) >> 34
	if hasWriter && !hasSeeker && !hasReader {
		numWriters := v >> 34
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
	const setR = PLOCK64_RL_1
	const maskR = PLOCK64_WL_ANY

	// Since all writes to this value are atomic, load is unnecessary,
	// but it makes the race detector happy
	if (atomic.LoadUint64(&p.lock) & maskR) != 0 {
		return false
	}
	if xadd64(&p.lock, setR)&maskR == 0 {
		return true
	}
	_ = subUint64(&p.lock, setR)

	return false
}

func (p *PMutex) RLock() {
	for {
		if p.tryRLock() {
			break
		}

		runtime.Gosched()
	}
}

func (p *PMutex) RUnlock() {
	const val = PLOCK64_RL_1
	subUint64(&p.lock, val)
}

//go:nosplit
func (p *PMutex) tryWLock() bool {
	const setR = PLOCK64_WL_1 | PLOCK64_SL_1 | PLOCK64_RL_1
	const maskR = PLOCK64_WL_ANY | PLOCK64_SL_ANY

	if xadd64(&p.lock, setR)&maskR == 0 {
		return true
	}
	_ = subUint64(&p.lock, setR)

	return false
}

func (p *PMutex) WLock() {
	const setR = PLOCK64_WL_1 | PLOCK64_SL_1 | PLOCK64_RL_1

	// acquire lock
	for {
		if p.tryWLock() {
			break
		}
		runtime.Gosched()
	}

	// wait for readers to leave
	for {
		if (atomic.LoadUint64(&p.lock) - setR) == 0 {
			break
		}
		// yield here in the this half acquired state;
		// this allows readers the opportunity to finish up, and prevents
		// new readers/writers from entering
		runtime.Gosched()
	}
}

func (p *PMutex) WUnlock() {
	const val = PLOCK64_WL_1 | PLOCK64_SL_1 | PLOCK64_RL_1
	subUint64(&p.lock, val)
}

//go:nosplit
func (p *PMutex) trySLock() bool {
	const setR = PLOCK64_SL_1 | PLOCK64_RL_1
	const maskR = PLOCK64_WL_ANY | PLOCK64_SL_ANY
	if xadd64(&p.lock, setR) == 0 {
		return true
	}
	subUint64(&p.lock, setR)
	return false
}

func (p *PMutex) SLock() {
	for {
		if p.trySLock() {
			break
		}
		runtime.Gosched()
	}
}

func (p *PMutex) SUnlock() {
	const val = PLOCK64_SL_1 + PLOCK64_RL_1
	subUint64(&p.lock, val)
}

func (p *PMutex) SToW() {
	t := xadd64(&p.lock, PLOCK64_WL_1)
	for {
		if t&PLOCK64_RL_ANY != PLOCK64_RL_1 {
			t = atomic.LoadUint64(&p.lock)
		}
	}
}

//go:nosplit
func (p *PMutex) tryALock() bool {
	const setR = PLOCK64_WL_1
	const maskR = PLOCK64_SL_ANY

	plr := atomic.LoadUint64(&p.lock) & maskR
	if plr == 0 {
		plr = xadd64(&p.lock, setR)
		for {
			if plr&maskR != 0 {
				_ = subUint64(&p.lock, setR)
				break
			}
			plr &= PLOCK64_RL_ANY
			if plr == 0 {
				break
			}
			plr = atomic.LoadUint64(&p.lock)
		}
	}

	return plr == 0
}

func (p *PMutex) ALock() {
	for {
		if p.tryALock() {
			break
		}
		runtime.Gosched()
	}
}

func (p *PMutex) AUnlock() {
	const val = PLOCK64_WL_1
	subUint64(&p.lock, val)
}

// Locker types

// RLocker
func (rw *PMutex) RLocker() sync.Locker {
	return (*rlocker)(rw)
}

type rlocker PMutex

func (r *rlocker) Lock()   { (*PMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*PMutex)(r).RUnlock() }

// WLocker
func (rw *PMutex) WLocker() sync.Locker {
	return (*wlocker)(rw)
}

type wlocker PMutex

func (r *wlocker) Lock()   { (*PMutex)(r).WLock() }
func (r *wlocker) Unlock() { (*PMutex)(r).WUnlock() }

// SLocker
func (rw *PMutex) SLocker() sync.Locker {
	return (*slocker)(rw)
}

type slocker PMutex

func (r *slocker) Lock()   { (*PMutex)(r).SLock() }
func (r *slocker) Unlock() { (*PMutex)(r).SUnlock() }
