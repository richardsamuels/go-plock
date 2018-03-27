package plock_test

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/richardsamuels/go-plock"
	"golang.org/x/net/context"
)

func TestPMutexCanObtainRLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	try(ctx, t, func() {
		m := plock.PMutex{}
		m.RLock()
		m.RUnlock()
	})
}

func TestPMutexCanObtainWLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	try(ctx, t, func() {
		m := plock.PMutex{}
		m.WLock()
		m.WUnlock()
	})
}

func printMutexState(ctx context.Context, m *plock.PMutex, d time.Duration) {
	for {
		select {
		case <-time.After(d):
			fmt.Println(m.String())
		case <-ctx.Done():
			return
		}
	}
}

func TestPMutexCanObtainWLockWithReaderContention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m := &plock.PMutex{}
	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	rs := rand.Intn(1000)
	for x := 0; x < rs; x++ {
		wg.Add(1)
		go func() {
			m.RLock()
			wg.Done()
			wg2.Wait()
			m.RUnlock()
		}()
	}

	try(ctx, t, func() {
		wg2.Done()
		wg.Wait()
		m.WLock()
		m.WUnlock()
	})
}
func TestPMutexCantObtainTwoWLocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d, _ := ctx.Deadline()
	ctx, cancel = context.WithDeadline(ctx, d)
	defer cancel()

	m := &plock.PMutex{}
	go printMutexState(ctx, m, 2*time.Second)

	for i := 0; i < 10; i++ {
		go func() {
			m.WLock()
			time.Sleep(time.Second)
			m.WLock()
			cancel()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Error("Expected this to timeout")
			}
			return
		}
	}
}

func try(ctx context.Context, t *testing.T, f func()) {
	d, _ := ctx.Deadline()
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	go func() {
		f()
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				t.Error("timed out")
			}
			return
		}
	}
}

func TestPMutexString(t *testing.T) {
	m := &plock.PMutex{}
	if !strings.HasSuffix(m.String(), "; U") {
		t.Error(m.String())
	}

	m.RLock()
	if !strings.HasSuffix(m.String(), "; R; readers: self only") {
		t.Error(m.String())
	}

	n := rand.Intn(1000) + 2
	for i := 2; i < n; i++ {
		m.RLock()
		suf := fmt.Sprintf("; R; readers: %d", i-1)
		if !strings.HasSuffix(m.String(), suf) {
			t.Errorf("expected %d readers, was: %s", i-1, m.String())
		}
	}
	m = &plock.PMutex{}

	m.WLock()
	if !strings.HasSuffix(m.String(), "; R+S+W; readers: self only") {
		t.Error(m.String())
	}

	m = &plock.PMutex{}
	m.RLock()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.WLock()
		wg.Done()
	}()
	time.Sleep(time.Second)
	if !strings.HasSuffix(m.String(), "; R+S+W; waiting for readers: 1") {
		t.Error(m.String())
	}
	m.RUnlock()

	//m2 := &plock.PMutex{}
	//if !strings.HasSuffix(m2.String(), "; A; writers: 1") {
	//	t.Error(m.String())
	//}
}
