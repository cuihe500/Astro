.PHONY: build run clean test lint swagger frontend-install frontend-run frontend-lint frontend-test frontend-build frontend-check

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
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs

frontend-install:
	cd web && npm install

frontend-run:
	cd web && npm run dev

frontend-lint:
	cd web && npm run lint

frontend-test:
	cd web && npm run test

frontend-build:
	cd web && npm run build

frontend-check: frontend-lint frontend-test frontend-build
