# go-plock
go-plock is an implementation of [progressive locks](http://wtarreau.blogspot.com/2018/02/progressive-locks-fast-upgradable.html)

## Architecture Support
Although files have been generated for all architectures support by Golang, this
project has not been tested under anything but x86_64. i386 has also been tested,
but only on x86_64 processors. In theory, if the tests pass on your architecture,
this should be safe.

## LICENSE
Portions of this project have been extracted/derived from Golang's source code. Namely:

* tests/from_rwmutex_test.go (derivative work)
* tests/rwmutex_test.go

These files remain under the same license as Golang, see tests/GOLANG_LICENSE.

The rest of this package is made available under the Apache 2.0 License, see LICENSE.
The full text of the license is available [here](http://www.apache.org/licenses/LICENSE-2.0)

While already covered by the Apache 2.0 License, this bears reiterating: This project comes
with no warranty, express or implied, nor any statement about fitness for purpose. You are
solely responsible for anything that happens as a result of your use of this library.
