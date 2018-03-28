// +build ignore

package plock

import (
	"fmt"
	"runtime"

	"sync/atomic"
)

type PMutex struct {
	lock uint64
}

const (
	leftShiftVal  = 0 //leftShiftVal
	rightShiftVal = 0 //rightShiftVal
)

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

	hasWriter := v&plock64WLAny != 0
	hasSeeker := v&plock64SLAny != 0
	hasReader := v&plock64RLAny != 0

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
	const setR = plock64RL1
	const maskR = plock64WLAny

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
	const val = plock64RL1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) tryRToA() bool {
	plr := atomic.LoadUint64(&p.lock) & plock64SLAny
	if plr == 0 {
		plr = xadd64(&p.lock, plock64WL1-plock64RL1)
		for {
			if plr&plock64SLAny != 0 {
				_ = subUint64(&p.lock, plock64WL1-plock64RL1)
				break
			}

			plr &= plock64RLAny
			if plr != 0 {
				break
			}
			plr = atomic.LoadUint64(&p.lock)
		}
	}

	return plr != 0
}

func (p *PMutex) RToA() {
	for {
		if p.tryRToA() {
			break
		}

		runtime.Gosched()
	}
}

func (p *PMutex) tryRToW() bool {
	const setR = plock64WL1 | plock64SL1
	const maskR = plock64WLAny | plock64SLAny
	var plr uint64
	for {
		plr = xadd64(&p.lock, setR)
		if plr&maskR != 0 {
			if xadd64(&p.lock, (^setR)+1) != 0 {
				break
			}
			continue
		}

		for plr != 0 {
			plr = atomic.LoadUint64(&p.lock) - (plock64WL1 | plock64SL1 | plock64RL1) //nolint:megacheck
			break
		}
	}

	return plr == 0
}

func (p *PMutex) RToW() {
	for {
		if p.tryRToW() {
			break
		}

		runtime.Gosched()
	}
}

func (p *PMutex) tryRToS() bool {
	plr := atomic.LoadUint64(&p.lock)
	if plr&(plock64WLAny|plock64SLAny) == 0 {
		plr = xadd64(&p.lock, plock64SL1) & (plock64WLAny | plock64SLAny)
		if plr != 0 {
			_ = subUint64(&p.lock, plock64SL1)
		}

	}

	return plr == 0
}

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
	const setR = plock64WL1 | plock64SL1 | plock64RL1
	const maskR = plock64WLAny | plock64SLAny

	if xadd64(&p.lock, setR)&maskR == 0 {
		return true
	}
	_ = subUint64(&p.lock, setR)

	return false
}

func (p *PMutex) WLock() {
	const setR = plock64WL1 | plock64SL1 | plock64RL1

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
	const val = plock64WL1 | plock64SL1 | plock64RL1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) WToR() {
	const val = plock64WL1 | plock64SL1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) WToS() {
	const val = plock64WL1
	_ = subUint64(&p.lock, val)
}

//go:nosplit
func (p *PMutex) trySLock() bool {
	const setR = plock64SL1 | plock64RL1
	const maskR = plock64WLAny | plock64SLAny
	if atomic.LoadUint64(&p.lock)&maskR == 0 {
		if xadd64(&p.lock, setR) == 0 {
			return true
		}
		_ = subUint64(&p.lock, setR)
	}
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
	const val = plock64SL1 + plock64RL1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) SToR() {
	const val = plock64SL1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) SToW() {
	t := xadd64(&p.lock, plock64WL1)
	for {
		if t&plock64RLAny != plock64RL1 {
			t = atomic.LoadUint64(&p.lock)
		}
	}
}

//go:nosplit
func (p *PMutex) tryALock() bool {
	const setR = plock64WL1
	const maskR = plock64SLAny

	plr := atomic.LoadUint64(&p.lock) & maskR
	if plr == 0 {
		plr = xadd64(&p.lock, setR)
		for {
			if plr&maskR != 0 {
				_ = subUint64(&p.lock, setR)
				break
			}
			plr &= plock64RLAny
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
	const val = plock64WL1
	_ = subUint64(&p.lock, val)
}
