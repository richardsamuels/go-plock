//go:generate go run ./templates/run.go ./templates/arch.go

package plock

const (
	plock32RL1   uint32 = 0x00000004
	plock32RLAny uint32 = 0x0000FFFC
	plock32SL1   uint32 = 0x00010000
	plock32SLAny uint32 = 0x00030000
	plock32WL1   uint32 = 0x00040000
	plock32WLAny uint32 = 0xFFFC0000
)

const (
	plock64RL1   uint64 = 0x0000000000000004
	plock64RLAny uint64 = 0x00000000FFFFFFFC
	plock64SL1   uint64 = 0x0000000100000000
	plock64SLAny uint64 = 0x0000000300000000
	plock64WL1   uint64 = 0x0000000400000000
	plock64WLAny uint64 = 0xFFFFFFFC00000000
)
