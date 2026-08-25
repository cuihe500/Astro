.PHONY: build run clean test fmt lint swagger legacy-inventory frontend-install frontend-ci frontend-run frontend-lint frontend-test frontend-build frontend-check docker-build docker-build-api docker-build-web workflow-lint git trellis trellis-context trellis-init trellis-session github governance-check

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

fmt:
	gofmt -w cmd internal pkg

lint:
	golangci-lint run

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs

legacy-inventory:
	@missing_tools=""; \
	if ! command -v mariadb >/dev/null 2>&1 && ! command -v mysql >/dev/null 2>&1; then missing_tools="$$missing_tools mariadb/mysql"; fi; \
	if ! command -v kubectl >/dev/null 2>&1; then missing_tools="$$missing_tools kubectl"; fi; \
	if [ -n "$$missing_tools" ]; then echo "缺少只读盘点客户端:$$missing_tools" >&2; exit 1; fi
	@test "$${ASTRO_RUNTIME_ENV:-}" = "test" || { echo "legacy-inventory 仅允许 ASTRO_RUNTIME_ENV=test" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_HOST:-}" || { echo "缺少 ASTRO_DATABASE_HOST" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_PORT:-}" || { echo "缺少 ASTRO_DATABASE_PORT" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_USER:-}" || { echo "缺少 ASTRO_DATABASE_USER" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_PASSWORD:-}" || { echo "缺少 ASTRO_DATABASE_PASSWORD" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_DBNAME:-}" || { echo "缺少 ASTRO_DATABASE_DBNAME" >&2; exit 1; }
	@test -r "$${ASTRO_KUBERNETES_KUBECONFIG:-}" || { echo "ASTRO_KUBERNETES_KUBECONFIG 不可读" >&2; exit 1; }
	@echo "数据库目标: $${ASTRO_DATABASE_HOST}:$${ASTRO_DATABASE_PORT}/$${ASTRO_DATABASE_DBNAME}"
	@database_client=mariadb; \
	if ! command -v "$$database_client" >/dev/null 2>&1; then database_client=mysql; fi; \
	if ! legacy_schema="$$(MYSQL_PWD="$$ASTRO_DATABASE_PASSWORD" "$$database_client" --protocol=TCP --connect-timeout=5 --host="$$ASTRO_DATABASE_HOST" --port="$$ASTRO_DATABASE_PORT" --user="$$ASTRO_DATABASE_USER" --database="$$ASTRO_DATABASE_DBNAME" --batch --skip-column-names --execute="SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'user_id') AND EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'namespace') AND NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'project_id')")"; then \
		echo "读取 apps schema 失败" >&2; \
		exit 1; \
	fi; \
	if [ "$$legacy_schema" = "1" ]; then \
		echo "旧 apps 活动记录（只读）:"; \
		MYSQL_PWD="$$ASTRO_DATABASE_PASSWORD" "$$database_client" --protocol=TCP --connect-timeout=5 --host="$$ASTRO_DATABASE_HOST" --port="$$ASTRO_DATABASE_PORT" --user="$$ASTRO_DATABASE_USER" --database="$$ASTRO_DATABASE_DBNAME" --table --execute="START TRANSACTION READ ONLY; SELECT COUNT(*) AS active_app_count FROM apps WHERE deleted_at IS NULL; SELECT id, name, user_id, namespace, status, created_at FROM apps WHERE deleted_at IS NULL ORDER BY id; COMMIT"; \
	elif [ "$$legacy_schema" = "0" ]; then \
		echo "未检测到旧 apps schema（需要 user_id、namespace 且没有 project_id）。"; \
	else \
		echo "无法判断 apps schema，返回值: $$legacy_schema" >&2; \
		exit 1; \
	fi
	@if ! context="$$(kubectl --kubeconfig="$$ASTRO_KUBERNETES_KUBECONFIG" config current-context)"; then \
		echo "读取 Kubernetes context 失败" >&2; \
		exit 1; \
	fi; \
	if [ -z "$$context" ]; then echo "Kubernetes context 为空" >&2; exit 1; fi; \
	echo "Kubernetes context: $$context"; \
	echo "旧 astro-user-* Namespace（只读，managed-by=astro）:"; \
	if ! resources="$$(kubectl --request-timeout=10s --kubeconfig="$$ASTRO_KUBERNETES_KUBECONFIG" get namespaces -l managed-by=astro -o name)"; then \
		echo "读取 Namespace 清单失败" >&2; \
		exit 1; \
	fi; \
	found=0; \
	for resource in $$resources; do \
		namespace="$${resource#namespace/}"; \
		suffix="$${namespace#astro-user-}"; \
		case "$$namespace:$$suffix" in astro-user-*:[0-9]*) \
			case "$$suffix" in *[!0-9]*) ;; *) echo "$$namespace"; found=1 ;; esac \
		;; esac; \
	done; \
	if [ "$$found" -eq 0 ]; then echo "（无）"; fi

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
