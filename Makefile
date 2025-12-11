.PHONY: build run clean swagger

APP_NAME=astro
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

lint:
	golangci-lint run

swagger:
	swag init -g cmd/server/main.go -o docs
