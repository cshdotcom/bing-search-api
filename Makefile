BINARY  := bing-search-api
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test vet fmt dist clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# 交叉编译全平台发行包到 dist/(linux/darwin/windows × amd64/arm64/386)
dist:
	bash build_release.sh $(VERSION)

clean:
	rm -rf $(BINARY) dist
