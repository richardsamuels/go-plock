package plock

import "sync"

// Locker interface just uses the W lock

// Unlock releases a Write lock
func (p *PMutex) Unlock() {
	p.WUnlock()
}

// Lock acquires a Write lock, waiting for current readers
func (p *PMutex) Lock() {
	p.WLock()
}

// RLocker
func (rw *PMutex) RLocker() RLocker {
	return (*rlocker)(rw)
}

type rlocker PMutex

func (r *rlocker) Lock()   { (*PMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*PMutex)(r).RUnlock() }
func (r *rlocker) RToW()   { (*PMutex)(r).RToW() }
func (r *rlocker) RToA()   { (*PMutex)(r).RToA() }
func (r *rlocker) RToS()   { (*PMutex)(r).RToS() }

type RLocker interface {
	// RToA upgrades an Read lock to an AtomicWrite lock
	RToA()
	// RToW upgrades an Read lock to an Write lock
	RToW()
	// RTos upgrades an Read lock to an Seek lock
	RToS()

	// Lock and Unlock can be used to acquire and release Read locks
	sync.Locker
}

// WLocker
func (rw *PMutex) WLocker() WLocker {
	return (*wlocker)(rw)
}

type wlocker PMutex

func (r *wlocker) Lock()   { (*PMutex)(r).WLock() }
func (r *wlocker) Unlock() { (*PMutex)(r).WUnlock() }
func (r *wlocker) WToR()   { (*PMutex)(r).WToR() }
func (r *wlocker) WToS()   { (*PMutex)(r).WToS() }

type WLocker interface {
	// WToR downgrades a Write lock to a Read lock
	WToR()
	// WToR downgrades a Write lock to a Seek lock
	WToS()

	// Lock and Unlock can be used to acquire and release Write locks
	sync.Locker
}

func (rw *PMutex) SLocker() SLocker {
	return (*slocker)(rw)
}

type slocker PMutex

func (r *slocker) Lock()   { (*PMutex)(r).SLock() }
func (r *slocker) Unlock() { (*PMutex)(r).SUnlock() }
func (r *slocker) SToR()   { (*PMutex)(r).SToR() }
func (r *slocker) SToW()   { (*PMutex)(r).SToW() }

type SLocker interface {
	// SToR downgrades a Seek lock to a Read lock
	SToR()
	// SToR upgrades a Seek lock to a Write lock
	SToW()

	// Lock and Unlock can be used to acquire and release Seek locks
	sync.Locker
}

// Lock and Unlock can be used to acquire and release AtomicWrite locks
func (rw *PMutex) ALocker() sync.Locker {
	return (*alocker)(rw)
}

type alocker PMutex

func (r *alocker) Lock()   { (*PMutex)(r).SLock() }
func (r *alocker) Unlock() { (*PMutex)(r).SUnlock() }

// vim: ft=go
