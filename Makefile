.PHONY: build run clean test lint swagger frontend-install frontend-ci frontend-run frontend-lint frontend-test frontend-build frontend-check docker-build docker-build-api docker-build-web workflow-lint

APP_NAME=astro
BUILD_DIR=bin
DOCKER_PLATFORM?=linux/arm64
IMAGE_REVISION?=local

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

frontend-ci:
	cd web && npm ci

frontend-run:
	cd web && npm run dev

frontend-lint:
	cd web && npm run lint

frontend-test:
	cd web && npm run test

frontend-build:
	cd web && npm run build

frontend-check: frontend-lint frontend-test frontend-build

docker-build: docker-build-api docker-build-web

docker-build-api:
	docker buildx build --platform $(DOCKER_PLATFORM) --load --build-arg VCS_REF=$(IMAGE_REVISION) -t astro-api:local -f Dockerfile .

docker-build-web:
	docker buildx build --platform $(DOCKER_PLATFORM) --load --build-arg VCS_REF=$(IMAGE_REVISION) -t astro-web:local -f web/Dockerfile .

workflow-lint:
	go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
