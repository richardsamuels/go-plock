// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	goBin := ""
	flag.StringVar(&goBin, "go", "go", "specify path to go binary")

	ret := 0
	wg := &sync.WaitGroup{}
	var max uint32 = uint32(runtime.NumCPU())
	var count uint32 = 0

	for OS, archs := range archsForOS {
		for _, arch := range archs {
			if _, ok := archWordSize[arch]; !ok {
				fmt.Fprintf(os.Stderr, "arch has no listed word size: %s", arch)
				ret = 1
				continue
			}

			for atomic.AddUint32(&count, 1) == max {
				atomic.AddUint32(&count, ^(uint32(0)))
				time.Sleep(1 * time.Second)
			}

			wg.Add(1)
			go func(OS, arch string) {
				defer wg.Done()
				output := "./lib/" + OS + "_" + arch
				cmd := exec.Command(goBin, "build")
				cmd.Env = append(cmd.Env,
					"GOOS="+OS,
					"GOARCH="+arch)
				cmd.Args = append(cmd.Args, "-o", output)
				var out bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &out

				fmt.Printf("Starting build for %s-%s\n", OS, arch)
				if err := cmd.Run(); err != nil {
					fmt.Printf("problem building %s: %v\nOutput:\n%s\n", output, err, string(out.Bytes()))
				}
				atomic.AddUint32(&count, ^(uint32(0)))
			}(OS, arch)
		}
	}

	wg.Wait()
	os.Exit(ret)
}
