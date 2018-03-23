package plock_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/richardsamuels/go-plock"
)

type locktestData struct {
	wgStart sync.WaitGroup
	wg      sync.WaitGroup

	m plock.PMutex

	shared     uint32
	readRatio  int
	step       uint32
	globalWork uint32

	finalWork uint32
	stopTime  time.Time
}

func locktest(thr uint, d *locktestData, t *testing.T) {
	loop := 0

	d.wgStart.Wait()

	for {
		d.m.RLock()
		if d.shared&(1<<thr) != 0 {
			t.Errorf("thr=%d, shared=%d : unexpected 1", thr, d.shared)
		}
		d.m.RUnlock()

		if (loop & 0xFF) >= d.readRatio {
			//d.m.WLock()
			d.shared |= (1 << thr)
			//d.m.WUnlock()

			//d.m.SLock()
			if (d.shared & (1 << thr)) == 0 {
				t.Errorf("thr=%d shared=0x%08x : unexpected 0\n", thr, d.shared)
			}
			//d.m.SToW()
			d.shared &= ^(1 << thr)
			//d.m.WUnlock()
		}

		d.m.RLock()
		if d.shared&(1<<thr) != 0 {
			t.Errorf("thr=%d, shared=%d : unexpected 1", thr, d.shared)
		}
		d.m.RUnlock()

		loop++
		if (loop & 0x7F) == 0 { /* don't access RAM too often */
			if atomic.AddUint32(&d.globalWork, 128) >= 20000000 {
				if atomic.AddUint32(&d.step, 1) == 3 {
					d.finalWork = atomic.LoadUint32(&d.globalWork)
					d.stopTime = time.Now()
				}
				break
			}
		}
	}

	fmt.Printf("%d done\n", thr)
	d.wg.Done()
}

func TestLock(t *testing.T) {
	d := locktestData{
		readRatio: 256,
		step:      2,
	}
	d.wgStart.Add(1)

	n := 2 * runtime.NumCPU()
	for i := 0; i < n; i++ {
		d.wg.Add(1)
		go locktest(uint(i), &d, t)
	}
	// trigger threads!
	start := time.Now()
	d.wgStart.Done()

	// wait for all work to complete
	d.wg.Wait()
	ms := time.Since(start) / time.Millisecond

	fmt.Printf("goroutines: %d loops: %d time(ms): %d\n",
		n, d.finalWork, ms)
}
