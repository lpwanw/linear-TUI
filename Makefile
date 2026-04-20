BINARY := lnr
PKG    := ./cmd/lnr

.PHONY: build run test vet lint install clean

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@command -v staticcheck >/dev/null && staticcheck ./... || echo "staticcheck not installed, skipping"

install:
	go install $(PKG)

clean:
	rm -f $(BINARY)
	rm -rf dist/
