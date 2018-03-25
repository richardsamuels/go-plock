//go:generate go run ./templates/run.go ./templates/arch.go

package plock

const (
	PLOCK32_RL_1   uint32 = 0x00000004
	PLOCK32_RL_ANY uint32 = 0x0000FFFC
	PLOCK32_SL_1   uint32 = 0x00010000
	PLOCK32_SL_ANY uint32 = 0x00030000
	PLOCK32_WL_1   uint32 = 0x00040000
	PLOCK32_WL_ANY uint32 = 0xFFFC0000
)

const (
	PLOCK64_RL_1   uint64 = 0x0000000000000004
	PLOCK64_RL_ANY uint64 = 0x00000000FFFFFFFC
	PLOCK64_SL_1   uint64 = 0x0000000100000000
	PLOCK64_SL_ANY uint64 = 0x0000000300000000
	PLOCK64_WL_1   uint64 = 0x0000000400000000
	PLOCK64_WL_ANY uint64 = 0xFFFFFFFC00000000
)