.PHONY: test lint run

test:
	go test ./...

lint:
	go fmt ./...
	go vet ./...

run:
	go run main.go
