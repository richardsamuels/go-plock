// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	bytes, err := ioutil.ReadFile("./templates/plock.go")
	if err != nil {
		panic(err)
	}

	if len(bytes) == 0 {
		panic("no bytes read")
	}

	for k, _ := range archWordSize {
		gen(bytes, k)
	}

}

func gen(bytes []byte, arch string) {
	b := make([]byte, len(bytes))
	for i := range bytes {
		b[i] = bytes[i]
	}

	out, err := os.Create(fmt.Sprintf("plockimpl_%s.go", arch))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	half := archWordSize[arch] / 2

	s := strings.Replace(string(b), "0 //leftShiftVal", fmt.Sprintf("%d", half), 1)
	s = strings.Replace(s, "0 //rightShiftVal", fmt.Sprintf("%d", half+2), 1)
	s = strings.Replace(s, "64", fmt.Sprintf("%d", archWordSize[arch]), -1)

	n, err := out.Write([]byte(s))
	if n != len([]byte(s)) {
		panic("didn't write correct # of bytes")
	}
	if err != nil {
		panic(err)
	}
}

type tmpl struct {
	LeftShift  uint
	RightShift uint
}
