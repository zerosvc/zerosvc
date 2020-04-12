# generate version number
version=$(shell git describe --tags --long --always --dirty|sed 's/^v//')

all:
	go build -ldflags "-X main.version=$(version)" timebot-client.go
	go build -ldflags "-X main.version=$(version)" timebot-server.go
	-@go fmt


version:
	@echo $(version)
