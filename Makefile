PREFIX ?= /usr/local
GO=go
NAME=nrk-spotify

all: fmt

fmt:
	gofmt -w=true *.go

build:
	mkdir -p bin
	$(GO) build -o bin/$(NAME)

install:
	cp -p bin/$(NAME) $(PREFIX)/bin/$(NAME)

deps:
	go get -u github.com/nitrous-io/goop
