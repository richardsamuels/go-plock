package plock

import "sync"

// Unlock releases a Write lock
func (p *PMutex) Unlock() {
	p.WUnlock()
}

// Lock acquires a Write lock, blocking until all current readers unlock
func (p *PMutex) Lock() {
	p.WLock()
}

// RLocker is a sync.Locker for acquiring/releasing Read Locks
func (rw *PMutex) RLocker() sync.Locker {
	return (*rlocker)(rw)
}

type rlocker PMutex

func (r *rlocker) Lock()   { (*PMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*PMutex)(r).RUnlock() }

// WLocker is a sync.Locker for acquiring/releasing Write Locks
func (rw *PMutex) WLocker() sync.Locker {
	return (*wlocker)(rw)
}

type wlocker PMutex

func (r *wlocker) Lock()   { (*PMutex)(r).WLock() }
func (r *wlocker) Unlock() { (*PMutex)(r).WUnlock() }

// SLocker is a sync.Locker for acquiring/releasing Seek Locks
func (rw *PMutex) SLocker() sync.Locker {
	return (*slocker)(rw)
}

type slocker PMutex

func (r *slocker) Lock()   { (*PMutex)(r).SLock() }
func (r *slocker) Unlock() { (*PMutex)(r).SUnlock() }

// ALocker creates a sync.Locker that can be used to acquire and release Atomic
// Write Locks
func (rw *PMutex) ALocker() sync.Locker {
	return (*alocker)(rw)
}

type alocker PMutex

func (r *alocker) Lock()   { (*PMutex)(r).SLock() }
func (r *alocker) Unlock() { (*PMutex)(r).SUnlock() }
