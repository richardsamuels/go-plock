package plock

import (
	"fmt"
	"testing"
)

func TestSubUint64(t *testing.T) {
	n := 100000

	var x uint64 = 5
	y := subUint64(&x, 4)
	if x != 1 || y != 1 {
		t.Error("basic test failed")
		t.FailNow()
	}

	for i := 0; i < n; i++ {
		base := randUint64()
		delta := randUint64()

		expected := base - delta
		msg := fmt.Sprintf("expected %d - %d to be %d", base, delta, expected)

		if expected != subUint64(&base, delta) || expected != base {
			t.Error(msg)
		}
	}
}

func TestXADD64(t *testing.T) {
	n := 100000

	var x uint64 = 5
	y := xadd64(&x, 4)
	if x != 9 || y != 5 {
		t.Error("basic test failed")
		t.FailNow()
	}

	for i := 0; i < n; i++ {
		base := randUint64()
		delta := randUint64()

		expected := base
		r := xadd64(&base, delta)
		msg := fmt.Sprintf("expected %d - %d to be %d, was %d", base, delta, expected, r)

		if expected != r || base != expected+delta {
			t.Error(msg)
		}
	}
}
