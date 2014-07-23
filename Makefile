PREFIX ?= /usr/local
GO=go
NAME=nrk-spotify

all: fmt

fmt:
	gofmt -w=true *.go

deps:
	$(GO) get -d -v

build:
	@mkdir -p bin
	$(GO) build -o bin/$(NAME)

install:
	cp -p bin/$(NAME) $(PREFIX)/bin/$(NAME)

test:
	$(GO) test
