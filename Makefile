PREFIX ?= /usr/local
NAME=nrk-spotify

all: test build

fmt:
	find . -maxdepth 2 -name '*.go' -exec gofmt -w=true {} \;

deps:
	@mkdir -p src/github.com/martinp
	@ln -sfn $(CURDIR) src/github.com/martinp/$(NAME)
	go get -d -v

build:
	@mkdir -p bin
	go build -o bin/$(NAME)

install:
	cp -p bin/$(NAME) $(PREFIX)/bin/$(NAME)

test:
	@find . -maxdepth 2 -name '*_test.go' -printf "%h\n" | xargs go test
