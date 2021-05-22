GOOS := linux
GOARCH := amd64

.PHONY: build

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -gcflags="all=-N -l" -o build/$(GOOS)-$(GOARCH) github.com/pnovotnak/daemonless
