// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// GOMAXPROCS=10 go test

package plock_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/richardsamuels/go-plock"
)

// There is a modified copy of this file in runtime/rwmutex_test.go.
// If you make any changes here, see if you should make them there.

func parallelReader(m *plock.PMutex, clocked, cunlock, cdone chan bool) {
	m.RLock()
	clocked <- true
	<-cunlock
	m.RUnlock()
	cdone <- true
}

func doTestParallelReaders(numReaders, gomaxprocs int) {
	runtime.GOMAXPROCS(gomaxprocs)
	var m plock.PMutex
	clocked := make(chan bool)
	cunlock := make(chan bool)
	cdone := make(chan bool)
	for i := 0; i < numReaders; i++ {
		go parallelReader(&m, clocked, cunlock, cdone)
	}
	// Wait for all parallel RLock()s to succeed.
	for i := 0; i < numReaders; i++ {
		<-clocked
	}
	for i := 0; i < numReaders; i++ {
		cunlock <- true
	}
	// Wait for the goroutines to finish.
	for i := 0; i < numReaders; i++ {
		<-cdone
	}
}

func TestParallelReaders(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	doTestParallelReaders(1, 4)
	doTestParallelReaders(3, 4)
	doTestParallelReaders(4, 2)
}

func reader(rwm *plock.PMutex, num_iterations int, activity *int32, cdone chan bool) {
	for i := 0; i < num_iterations; i++ {
		rwm.RLock()
		n := atomic.AddInt32(activity, 1)
		if n < 1 || n >= 10000 {
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		for i := 0; i < 100; i++ {
		}
		atomic.AddInt32(activity, -1)
		rwm.RUnlock()
	}
	cdone <- true
}

func writer(rwm *plock.PMutex, num_iterations int, activity *int32, cdone chan bool) {
	for i := 0; i < num_iterations; i++ {
		rwm.Lock()
		n := atomic.AddInt32(activity, 10000)
		if n != 10000 {
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		for i := 0; i < 100; i++ {
		}
		atomic.AddInt32(activity, -10000)
		rwm.Unlock()
	}
	cdone <- true
}

func HammerPMutex(gomaxprocs, numReaders, num_iterations int) {
	runtime.GOMAXPROCS(gomaxprocs)
	// Number of active readers + 10000 * number of active writers.
	var activity int32
	var rwm plock.PMutex
	cdone := make(chan bool)
	go writer(&rwm, num_iterations, &activity, cdone)
	var i int
	for i = 0; i < numReaders/2; i++ {
		go reader(&rwm, num_iterations, &activity, cdone)
	}
	go writer(&rwm, num_iterations, &activity, cdone)
	for ; i < numReaders; i++ {
		go reader(&rwm, num_iterations, &activity, cdone)
	}
	// Wait for the 2 writers and all readers to finish.
	for i := 0; i < 2+numReaders; i++ {
		<-cdone
	}
}

func TestPMutex(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	n := 1000
	if testing.Short() {
		n = 5
	}
	HammerPMutex(1, 1, n)
	HammerPMutex(1, 3, n)
	HammerPMutex(1, 10, n)
	HammerPMutex(4, 1, n)
	HammerPMutex(4, 3, n)
	HammerPMutex(4, 10, n)
	HammerPMutex(10, 1, n)
	HammerPMutex(10, 3, n)
	HammerPMutex(10, 10, n)
	HammerPMutex(10, 5, n)
}

func TestRLocker(t *testing.T) {
	var wl plock.PMutex
	var rl sync.Locker
	wlocked := make(chan bool, 1)
	rlocked := make(chan bool, 1)
	rl = wl.RLocker()
	n := 10
	go func() {
		for i := 0; i < n; i++ {
			rl.Lock()
			rl.Lock()
			rlocked <- true
			wl.Lock()
			wlocked <- true
		}
	}()
	for i := 0; i < n; i++ {
		<-rlocked
		rl.Unlock()
		select {
		case <-wlocked:
			t.Fatal("RLocker() didn't read-lock it")
		default:
		}
		rl.Unlock()
		<-wlocked
		select {
		case <-rlocked:
			t.Fatal("RLocker() didn't respect the write lock")
		default:
		}
		wl.Unlock()
	}
}

func BenchmarkPMutexUncontended(b *testing.B) {
	type PaddedPMutex struct {
		plock.PMutex
		pad [32]uint32
	}
	b.RunParallel(func(pb *testing.PB) {
		var rwm PaddedPMutex
		for pb.Next() {
			rwm.RLock()
			rwm.RLock()
			rwm.RUnlock()
			rwm.RUnlock()
			rwm.Lock()
			rwm.Unlock()
		}
	})
}

func benchmarkPMutex(b *testing.B, localWork, writeRatio int) {
	var rwm plock.PMutex
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			foo++
			if foo%writeRatio == 0 {
				rwm.Lock()
				rwm.Unlock()
			} else {
				rwm.RLock()
				for i := 0; i != localWork; i += 1 {
					foo *= 2
					foo /= 2
				}
				rwm.RUnlock()
			}
		}
		_ = foo
	})
}

func BenchmarkPMutexWrite100(b *testing.B) {
	benchmarkPMutex(b, 0, 100)
}

func BenchmarkPMutexWrite10(b *testing.B) {
	benchmarkPMutex(b, 0, 10)
}

func BenchmarkPMutexWorkWrite100(b *testing.B) {
	benchmarkPMutex(b, 100, 100)
}

func BenchmarkPMutexWorkWrite10(b *testing.B) {
	benchmarkPMutex(b, 100, 10)
}
