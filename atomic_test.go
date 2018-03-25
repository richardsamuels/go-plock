package plock

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	fmt.Printf("atomic_test.go: %s seed: %d\n", randomType, seed)
}
