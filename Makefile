VERSION ?= 0.0.1
REVISION = $(shell git rev-parse --short HEAD)
LDFLAGS = -X 'main.version=$(VERSION)' -X 'main.revision=$(REVISION)' -w -s

all: lint build

lint:
	@golangci-lint run --tests --disable-all --enable=goimports --enable=golint --enable=govet --enable=errcheck --enable=staticcheck

build:
	CGO_ENABLED=1 go build -buildmode=c-shared -o build/out_slack_ex.so -trimpath -ldflags "$(LDFLAGS)" .

clean:
	@go clean
	@rm -rf build
