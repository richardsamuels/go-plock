all: build test

build:
	go build .

test:
	go test -race -test.v ./tests
	go test -test.v ./tests

