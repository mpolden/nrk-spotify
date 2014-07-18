PREFIX ?= /usr/local
GO=go
NAME=nrk-spotify

all: fmt test

fmt:
	gofmt -w=true *.go

run:
	$(GO) run util.go main.go

build:
	mkdir -p bin
	$(GO) build -o bin/$(NAME)

install:
	cp -p bin/$(NAME) $(PREFIX)/bin/$(NAME)

test:
	$(GO) test
