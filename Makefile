NAME=nrk-spotify

all: deps test build

fmt:
	go fmt ./...

test:
	go test ./...

deps:
	go get -d -v ./...

install:
	go install

build:
	@mkdir -p bin
	go build -o bin/$(NAME)

docker-image:
	docker build -t martinp/nrk-spotify .
