// +build ignore

package main

var archWordSize = map[string]uint{
	"arm":      32,
	"arm64":    64,
	"386":      32,
	"amd64":    64,
	"ppc64":    64,
	"ppc64le":  64,
	"mips":     32,
	"mipsle":   32,
	"mips64":   64,
	"mips64le": 64,
	"s390x":    64,
}

var archsForOS = map[string][]string{
	"android":   []string{"arm"},
	"darwin":    []string{"386", "amd64", "arm", "arm64"},
	"dragonfly": []string{"amd64"},
	"freebsd":   []string{"386", "amd64", "arm"},
	"linux":     []string{"386", "amd64", "arm", "arm64", "ppc64", "ppc64le", "mips", "mipsle", "mips64", "mips64le", "s390x"},
	"netbsd":    []string{"386", "amd64", "arm"},
	"openbsd":   []string{"386", "amd64", "arm"},
	"plan9":     []string{"386", "amd64"},
	"solaris":   []string{"amd64"},
	"windows":   []string{"386", "amd64"},
}
