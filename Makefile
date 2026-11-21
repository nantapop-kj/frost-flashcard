VERSION ?= dev
export VERSION

.PHONY: run build test check app dmg windows icons tidy clean

run:
	go run -tags private_mac_apis ./cmd

build:
	go build -tags private_mac_apis -o build/bin/frost-flashcard ./cmd

test:
	go test ./...

check:
	@test -z "$$(gofmt -l .)" || (echo "not gofmt-ed:"; gofmt -l .; exit 1)
	go vet ./...
	go test -race ./...

app:
	./scripts/build-mac.sh

dmg:
	./scripts/build-mac.sh --dmg

windows:
	./scripts/build-windows.sh

icons:
	./scripts/make-icons.sh

tidy:
	go mod tidy

clean:
	rm -rf build/bin
