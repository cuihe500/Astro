.PHONY: build run clean test lint swagger frontend-install frontend-ci frontend-run frontend-lint frontend-test frontend-build frontend-check docker-build docker-build-api docker-build-web workflow-lint git trellis trellis-context trellis-init trellis-session github governance-check

APP_NAME=astro
BUILD_DIR=bin
DOCKER_PLATFORM?=linux/arm64
IMAGE_REVISION?=local
TRELLIS_ARGS?=
TRELLIS_CONTEXT_ARGS?=
TRELLIS_INIT_ARGS?=
TRELLIS_SESSION_ARGS?=
GITHUB_ARGS?=
GIT_ARGS?=
TASK?=

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

git:
	git $(GIT_ARGS)

trellis:
	python3 ./.trellis/scripts/task.py $(TRELLIS_ARGS)

trellis-context:
	python3 ./.trellis/scripts/get_context.py $(TRELLIS_CONTEXT_ARGS)

trellis-init:
	python3 ./.trellis/scripts/init_developer.py $(TRELLIS_INIT_ARGS)

trellis-session:
	python3 ./.trellis/scripts/add_session.py $(TRELLIS_SESSION_ARGS)

github:
	gh $(GITHUB_ARGS)

governance-check:
	$(if $(TASK),,$(error 缺少 TASK=<Trellis任务目录>))
	TASK_DIR="$(TASK)" python3 -c 'import json, os, pathlib, re; paths = [pathlib.Path(".github/ISSUE_TEMPLATE") / name for name in ("feature.yml", "bug.yml", "maintenance.yml")]; forms = [path.read_text(encoding="utf-8") for path in paths]; assert all(text.startswith("name: ") and "\ndescription: " in text and "\nprojects: [\"cuihe500/6\"]\n" in text and "\nbody:\n" in text and "\t" not in text for text in forms); config = pathlib.Path(".github/ISSUE_TEMPLATE/config.yml").read_text(encoding="utf-8"); assert "blank_issues_enabled: false" in config and "https://github.com/cuihe500/Astro/security/advisories/new" in config; assert pathlib.Path(".github/pull_request_template.md").is_file() and pathlib.Path("docs/development-workflow.md").is_file(); task = json.loads((pathlib.Path(os.environ["TASK_DIR"]) / "task.json").read_text(encoding="utf-8")); meta = task.get("meta", {}); issue_pattern = re.compile(r"https://github\.com/cuihe500/Astro/issues/[1-9][0-9]*"); assert issue_pattern.fullmatch(meta.get("github_issue", "")) and meta.get("github_project") == "https://github.com/users/cuihe500/projects/6"; print("治理配置校验通过")'
