.PHONY: build test clean

build:
	mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/kraken-bot ./cmd/kraken-bot

test:
	go test ./...

clean:
	rm -f dist/kraken-bot
