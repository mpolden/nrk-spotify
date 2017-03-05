all: deps test install vet

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet ./...

deps:
	go get -d -v ./...

install:
	go install
