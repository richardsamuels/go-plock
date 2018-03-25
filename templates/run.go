// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func main() {
	bytes, err := ioutil.ReadFile("./templates/plock.go.tmpl")
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

	s := strings.Replace(string(b), "64", fmt.Sprintf("%d", archWordSize[arch]), -1)
	temp := template.Must(template.New("").Parse(s))
	temp.Execute(out, tmpl{
		LeftShift:  (archWordSize[arch]) / 2,
		RightShift: (archWordSize[arch] / 2) + 2,
	})
}

type tmpl struct {
	LeftShift  uint
	RightShift uint
}
