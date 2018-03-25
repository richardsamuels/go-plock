package plock

import (
	"fmt"
	"runtime"

	"sync/atomic"
)

type PMutex struct {
    lock uint64
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
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) tryRToA() bool {
	plr := atomic.LoadUint64(&p.lock) & PLOCK64_SL_ANY
	if plr == 0 {
		plr = xadd64(&p.lock, PLOCK64_WL_1-PLOCK64_RL_1)
		for {
			if plr&PLOCK64_SL_ANY != 0 {
				_ = subUint64(&p.lock, PLOCK64_WL_1-PLOCK64_RL_1)
				break
			}

			plr &= PLOCK64_RL_ANY
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
	const setR = PLOCK64_WL_1 | PLOCK64_SL_1
	const maskR = PLOCK64_WL_ANY | PLOCK64_SL_ANY
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
			plr = atomic.LoadUint64(&p.lock) - (PLOCK64_WL_1 | PLOCK64_SL_1 | PLOCK64_RL_1) //nolint:megacheck
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
	if plr&(PLOCK64_WL_ANY|PLOCK64_SL_ANY) == 0 {
		plr = xadd64(&p.lock, PLOCK64_SL_1) & (PLOCK64_WL_ANY | PLOCK64_SL_ANY)
		if plr != 0 {
			_ = subUint64(&p.lock, PLOCK64_SL_1)
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
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) WToR() {
	const val = PLOCK64_WL_1 | PLOCK64_SL_1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) WToS() {
	const val = PLOCK64_WL_1
	_ = subUint64(&p.lock, val)
}

//go:nosplit
func (p *PMutex) trySLock() bool {
	const setR = PLOCK64_SL_1 | PLOCK64_RL_1
	const maskR = PLOCK64_WL_ANY | PLOCK64_SL_ANY
	if atomic.LoadUint64(&p.lock) & maskR == 0  {
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
	const val = PLOCK64_SL_1 + PLOCK64_RL_1
	_ = subUint64(&p.lock, val)
}

func (p *PMutex) SToR() {
	const val = PLOCK64_SL_1
	_ = subUint64(&p.lock, val)
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
	_ = subUint64(&p.lock, val)
}

// vim: ft=go
