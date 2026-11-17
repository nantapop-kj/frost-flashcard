.PHONY: run build tidy

run:
	go run ./cmd

build:
	go build -o bin/frost-flashcard ./cmd

tidy:
	go mod tidy
