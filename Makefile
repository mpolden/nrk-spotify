PREFIX ?= /usr/local
GO=go
NAME=nrk-spotify

all: fmt

fmt:
	gofmt -w=true *.go

build:
	$(GO) build -o nrk-spotify

install:
	cp -p bin/$(NAME) $(PREFIX)/bin/$(NAME)

test:
	go test
