// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// GOMAXPROCS=10 go test

package plock_test

import (
	. "sync"
	"testing"
)

func BenchmarkRWMutexUncontended(b *testing.B) {
	type PaddedRWMutex struct {
		RWMutex
		pad [32]uint32
	}
	b.RunParallel(func(pb *testing.PB) {
		var rwm PaddedRWMutex
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

func benchmarkRWMutex(b *testing.B, localWork, writeRatio int) {
	var rwm RWMutex
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

func BenchmarkRWMutexWrite100(b *testing.B) {
	benchmarkRWMutex(b, 0, 100)
}

func BenchmarkRWMutexWrite10(b *testing.B) {
	benchmarkRWMutex(b, 0, 10)
}

func BenchmarkRWMutexWorkWrite100(b *testing.B) {
	benchmarkRWMutex(b, 100, 100)
}

func BenchmarkRWMutexWorkWrite10(b *testing.B) {
	benchmarkRWMutex(b, 100, 10)
}
