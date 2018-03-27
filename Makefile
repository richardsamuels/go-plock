all: build test

lint:
	go get -u github.com/alecthomas/gometalinter
	${GOPATH}/bin/gometalinter . --disable="golint" --disable="gocyclo" --disable="goconst"

isuptodate:
	go generate
	git diff --exit-code "plockimpl_*.go" > /dev/null 2>/dev/null

build:
	go generate
	go build .

test:
	GOCACHE=off go test -test.v -race ./...
	GOCACHE=off go test -test.v ./...

test-short:
	GOCACHE=off go test -test.v -test.short -race ./...
	GOCACHE=off go test -test.v -test.short  ./...


short: build test-short

bench: build
	GOCACHE=off go test -test.v -bench ./...

clean:
	rm -f plockimpl_*.go
	rm -rf lib

xc:
	go generate
	mkdir lib
	go run ./templates/xc.go ./templates/arch.go

.PHONY: clean xc
