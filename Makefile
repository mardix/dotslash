VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE   ?= ghcr.io/mardix/dotslash

.PHONY: run build test image icons fmt clean

## run: start locally on :8080 with password "admin" (data in ./data)
run:
	DS_PASSWORD=admin go run .

## build: build the binary for this machine
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dotslash .

## test: Go unit tests (permissions, sessions, users, import/export, backups)
test:
	go test ./...

## image: build the Docker image locally
image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

## icons: refresh icons.json from the Dashboard Icons repository
icons:
	./scripts/update-icons.sh

fmt:
	gofmt -w .

clean:
	rm -f dotslash
