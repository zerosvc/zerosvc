# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')
binfile=catalogd

all:
	rm -rf _vendor
	go build -ldflags "-X main.version=$(version)" $(binfile).go
	-@go fmt


arm:
	GOARCH=arm go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm $(binfile).go
	GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm64 $(binfile).go
version:
	@echo $(version)
