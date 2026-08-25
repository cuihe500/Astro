.PHONY: build run clean test test-integration fmt lint swagger legacy-inventory legacy-delete-namespace frontend-install frontend-ci frontend-run frontend-lint frontend-test frontend-build frontend-check docker-build docker-build-api docker-build-web workflow-lint git trellis trellis-context trellis-init trellis-session github governance-check

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

test-integration:
	@test "$${ASTRO_RUNTIME_ENV:-}" = "test" || { echo "test-integration 仅允许 ASTRO_RUNTIME_ENV=test" >&2; exit 1; }
	@test "$${ASTRO_MARIADB_INTEGRATION:-}" = "1" || { echo "必须显式设置 ASTRO_MARIADB_INTEGRATION=1" >&2; exit 1; }
	go test -count=1 -tags=integration -v ./internal/repository

fmt:
	gofmt -w cmd internal pkg

lint:
	golangci-lint run

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs

legacy-inventory:
	@test "$${ASTRO_RUNTIME_ENV:-}" = "test" || { echo "legacy-inventory 仅允许 ASTRO_RUNTIME_ENV=test" >&2; exit 1; }
	@test -n "$${ASTRO_DATABASE_PORT:-}" || { echo "缺少 ASTRO_DATABASE_PORT" >&2; exit 1; }
	@case "$$ASTRO_DATABASE_PORT" in *[!0-9]*|0*) echo "ASTRO_DATABASE_PORT 必须是有效端口" >&2; exit 1;; esac; \
	if [ "$${#ASTRO_DATABASE_PORT}" -gt 5 ] || [ "$$ASTRO_DATABASE_PORT" -gt 65535 ]; then echo "ASTRO_DATABASE_PORT 必须是有效端口" >&2; exit 1; fi
	@missing_tools=""; \
	if ! command -v docker >/dev/null 2>&1; then missing_tools="$$missing_tools docker"; fi; \
	if ! command -v kubectl >/dev/null 2>&1; then missing_tools="$$missing_tools kubectl"; fi; \
	if [ -n "$$missing_tools" ]; then echo "缺少只读盘点客户端:$$missing_tools" >&2; exit 1; fi
	@if ! running_containers="$$(docker ps --format '{{.ID}}')"; then \
		echo "读取 Docker 容器失败" >&2; \
		exit 1; \
	fi; \
	database_containers=""; \
	for container in $$running_containers; do \
		if ! published_ports="$$(docker port "$$container")"; then echo "读取 Docker 端口映射失败" >&2; exit 1; fi; \
		if printf '%s\n' "$$published_ports" | grep -Eq ":$$ASTRO_DATABASE_PORT$$"; then database_containers="$$database_containers $$container"; fi; \
	done; \
	set -- $$database_containers; \
	if [ "$$#" -ne 1 ]; then \
		echo "宿主端口 $$ASTRO_DATABASE_PORT 必须恰好由一个运行中的 Docker 容器发布，当前为 $$# 个" >&2; \
		exit 1; \
	fi; \
	database_container="$$1"; \
	if ! database_container_name="$$(docker ps --filter "id=$$database_container" --format '{{.Names}}')" || [ -z "$$database_container_name" ]; then \
		echo "读取数据库容器名称失败" >&2; \
		exit 1; \
	fi; \
	echo "数据库容器: $$database_container_name（宿主端口 $$ASTRO_DATABASE_PORT）"; \
	database_query() { \
		docker exec "$$database_container" sh -eu -c '\
			database_client=mariadb; \
			if ! command -v "$$database_client" >/dev/null 2>&1; then database_client=mysql; fi; \
			command -v "$$database_client" >/dev/null 2>&1 || { echo "数据库容器缺少 mariadb/mysql 客户端" >&2; exit 1; }; \
			database_name=$${MARIADB_DATABASE:-$${MYSQL_DATABASE:-}}; \
			database_user=$${MARIADB_USER:-$${MYSQL_USER:-}}; \
			if [ -n "$$database_user" ]; then \
				database_password=$${MARIADB_PASSWORD:-$${MYSQL_PASSWORD:-}}; \
				database_password_file=$${MARIADB_PASSWORD_FILE:-$${MYSQL_PASSWORD_FILE:-}}; \
			else \
				database_user=root; \
				database_password=$${MARIADB_ROOT_PASSWORD:-$${MYSQL_ROOT_PASSWORD:-}}; \
				database_password_file=$${MARIADB_ROOT_PASSWORD_FILE:-$${MYSQL_ROOT_PASSWORD_FILE:-}}; \
			fi; \
			[ -n "$$database_name" ] || { echo "数据库容器缺少数据库名称环境变量" >&2; exit 1; }; \
			if [ -z "$$database_password" ]; then \
				[ -n "$$database_password_file" ] && [ -r "$$database_password_file" ] || { echo "数据库容器缺少可读密码环境" >&2; exit 1; }; \
				if ! database_password="$$(cat "$$database_password_file")"; then echo "读取数据库容器密码文件失败" >&2; exit 1; fi; \
			fi; \
			[ -n "$$database_password" ] || { echo "数据库容器密码为空" >&2; exit 1; }; \
			if [ "$$2" = scalar ]; then \
				MYSQL_PWD="$$database_password" "$$database_client" --protocol=socket --connect-timeout=5 --user="$$database_user" --database="$$database_name" --batch --skip-column-names --execute="$$1"; \
			else \
				MYSQL_PWD="$$database_password" "$$database_client" --protocol=socket --connect-timeout=5 --user="$$database_user" --database="$$database_name" --table --execute="$$1"; \
			fi' sh "$$1" "$$2"; \
	}; \
	if ! legacy_schema="$$(database_query "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'user_id') AND EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'namespace') AND NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'apps' AND column_name = 'project_id')" scalar)"; then \
		echo "读取 apps schema 失败" >&2; \
		exit 1; \
	fi; \
	if [ "$$legacy_schema" = "1" ]; then \
		echo "旧 apps 活动记录（只读）:"; \
		if ! database_query "SELECT COUNT(*) AS active_app_count FROM apps WHERE deleted_at IS NULL; SELECT id, name, user_id, namespace, status, created_at FROM apps WHERE deleted_at IS NULL ORDER BY id" table; then echo "读取旧 apps 记录失败" >&2; exit 1; fi; \
	elif [ "$$legacy_schema" = "0" ]; then \
		echo "未检测到旧 apps schema（需要 user_id、namespace 且没有 project_id）。"; \
	else \
		echo "无法判断 apps schema，返回值: $$legacy_schema" >&2; \
		exit 1; \
	fi
	@kubeconfig="$${ASTRO_KUBERNETES_KUBECONFIG:-}"; \
	kind_kubeconfig=""; \
	if [ -n "$$kubeconfig" ]; then \
		[ -r "$$kubeconfig" ] || { echo "ASTRO_KUBERNETES_KUBECONFIG 不可读" >&2; exit 1; }; \
		export KUBECONFIG="$$kubeconfig"; \
	fi; \
	if context="$$(kubectl config current-context 2>/dev/null)" && [ -n "$$context" ]; then \
		:; \
	elif [ -n "$$kubeconfig" ]; then \
		echo "读取 Kubernetes context 失败" >&2; \
		exit 1; \
	else \
		command -v kind >/dev/null 2>&1 || { echo "kubectl 默认 context 不可用且缺少 kind 客户端" >&2; exit 1; }; \
		if ! kind_clusters="$$(kind get clusters)"; then echo "读取本机 kind 集群失败" >&2; exit 1; fi; \
		set -- $$kind_clusters; \
		if [ "$$#" -ne 1 ]; then echo "kubectl 默认 context 不可用时本机必须恰好有一个 kind 集群，当前为 $$# 个" >&2; exit 1; fi; \
		if ! kind_kubeconfig="$$(kind get kubeconfig --name "$$1")"; then echo "读取 kind kubeconfig 失败" >&2; exit 1; fi; \
		if ! context="$$(printf '%s\n' "$$kind_kubeconfig" | kubectl --kubeconfig=/dev/stdin config current-context 2>/dev/null)"; then echo "读取 kind context 失败" >&2; exit 1; fi; \
	fi; \
	case "$$context" in kind-*) ;; *) echo "Kubernetes context 必须是 kind-*，当前为 $${context:-空}" >&2; exit 1;; esac; \
	kubectl_get() { \
		if [ -n "$$kind_kubeconfig" ]; then \
			printf '%s\n' "$$kind_kubeconfig" | kubectl --kubeconfig=/dev/stdin --context="$$context" --request-timeout=10s get "$$@"; \
		else \
			kubectl --context="$$context" --request-timeout=10s get "$$@"; \
		fi; \
	}; \
	echo "Kubernetes context: $$context"; \
	echo "旧 astro-user-* Namespace（只读，managed-by=astro）:"; \
	if ! resources="$$(kubectl_get namespaces -l managed-by=astro -o name)"; then echo "读取 Namespace 清单失败" >&2; exit 1; fi; \
	found=0; \
	for resource in $$resources; do \
		namespace="$${resource#namespace/}"; \
		suffix="$${namespace#astro-user-}"; \
		case "$$namespace:$$suffix" in astro-user-*:[0-9]*) \
			case "$$suffix" in \
				*[!0-9]*) ;; \
				*) \
					echo "$$namespace"; \
					if ! namespace_resources="$$(kubectl_get deployments,services,pods --namespace="$$namespace" -o name)"; then echo "读取 Namespace $$namespace 的资源清单失败" >&2; exit 1; fi; \
					if [ -z "$$namespace_resources" ]; then \
						echo "  Deployment/Service/Pod: 空"; \
					else \
						echo "  Deployment/Service/Pod: 非空"; \
						printf '  %s\n' $$namespace_resources; \
					fi; \
					found=1 \
				;; \
			esac \
		;; esac; \
	done; \
	if [ "$$found" -eq 0 ]; then echo "（无）"; fi

legacy-delete-namespace:
	@test "$${ASTRO_RUNTIME_ENV:-}" = "test" || { echo "legacy-delete-namespace 仅允许 ASTRO_RUNTIME_ENV=test" >&2; exit 1; }
	@test -n "$${LEGACY_NAMESPACE:-}" || { echo "必须显式传入 LEGACY_NAMESPACE" >&2; exit 1; }
	@namespace="$$LEGACY_NAMESPACE"; \
	suffix="$${namespace#astro-user-}"; \
	case "$$namespace:$$suffix" in astro-user-*:[0-9]*) case "$$suffix" in *[!0-9]*) echo "LEGACY_NAMESPACE 必须严格匹配 astro-user-<数字>" >&2; exit 1;; esac ;; *) echo "LEGACY_NAMESPACE 必须严格匹配 astro-user-<数字>" >&2; exit 1;; esac
	@command -v kubectl >/dev/null 2>&1 || { echo "缺少 kubectl" >&2; exit 1; }
	@namespace="$$LEGACY_NAMESPACE"; \
	kubeconfig="$${ASTRO_KUBERNETES_KUBECONFIG:-}"; \
	kind_kubeconfig=""; \
	if [ -n "$$kubeconfig" ]; then \
		[ -r "$$kubeconfig" ] || { echo "ASTRO_KUBERNETES_KUBECONFIG 不可读" >&2; exit 1; }; \
		export KUBECONFIG="$$kubeconfig"; \
	fi; \
	if context="$$(kubectl config current-context 2>/dev/null)" && [ -n "$$context" ]; then \
		:; \
	elif [ -n "$$kubeconfig" ]; then \
		echo "读取 Kubernetes context 失败" >&2; \
		exit 1; \
	else \
		command -v kind >/dev/null 2>&1 || { echo "kubectl 默认 context 不可用且缺少 kind 客户端" >&2; exit 1; }; \
		if ! kind_clusters="$$(kind get clusters)"; then echo "读取本机 kind 集群失败" >&2; exit 1; fi; \
		set -- $$kind_clusters; \
		if [ "$$#" -ne 1 ]; then echo "kubectl 默认 context 不可用时本机必须恰好有一个 kind 集群，当前为 $$# 个" >&2; exit 1; fi; \
		if ! kind_kubeconfig="$$(kind get kubeconfig --name "$$1")"; then echo "读取 kind kubeconfig 失败" >&2; exit 1; fi; \
		if ! context="$$(printf '%s\n' "$$kind_kubeconfig" | kubectl --kubeconfig=/dev/stdin config current-context 2>/dev/null)"; then echo "读取 kind context 失败" >&2; exit 1; fi; \
	fi; \
	case "$$context" in kind-*) ;; *) echo "Kubernetes context 必须是 kind-*，当前为 $${context:-空}" >&2; exit 1;; esac; \
	kubectl_get() { \
		if [ -n "$$kind_kubeconfig" ]; then \
			printf '%s\n' "$$kind_kubeconfig" | kubectl --kubeconfig=/dev/stdin --context="$$context" --request-timeout=10s get "$$@"; \
		else \
			kubectl --context="$$context" --request-timeout=10s get "$$@"; \
		fi; \
	}; \
	kubectl_delete() { \
		if [ -n "$$kind_kubeconfig" ]; then \
			printf '%s\n' "$$kind_kubeconfig" | kubectl --kubeconfig=/dev/stdin --context="$$context" --request-timeout=70s delete "$$@"; \
		else \
			kubectl --context="$$context" --request-timeout=70s delete "$$@"; \
		fi; \
	}; \
	if ! resolved_namespace="$$(kubectl_get namespace "$$namespace" -o name)" || [ "$$resolved_namespace" != "namespace/$$namespace" ]; then echo "目标 Namespace 不存在或无法精确解析: $$namespace" >&2; exit 1; fi; \
	if ! managed_by="$$(kubectl_get namespace "$$namespace" -o jsonpath='{.metadata.labels.managed-by}')"; then echo "读取 Namespace 标签失败" >&2; exit 1; fi; \
	if [ "$$managed_by" != "astro" ]; then echo "Namespace 缺少 managed-by=astro 标签，拒绝删除" >&2; exit 1; fi; \
	if ! namespace_resources="$$(kubectl_get deployments,services,pods --namespace="$$namespace" -o name)"; then echo "读取 Namespace 资源清单失败" >&2; exit 1; fi; \
	if [ -n "$$namespace_resources" ]; then echo "Namespace 非空，拒绝删除:" >&2; printf '  %s\n' $$namespace_resources >&2; exit 1; fi; \
	echo "删除空 Namespace: $$namespace（context: $$context）"; \
	if ! kubectl_delete namespace "$$namespace" --wait=true --timeout=60s; then echo "删除 Namespace 失败" >&2; exit 1; fi; \
	if ! remaining="$$(kubectl_get namespace "$$namespace" --ignore-not-found -o name)"; then echo "验证 Namespace 删除结果失败" >&2; exit 1; fi; \
	if [ -n "$$remaining" ]; then echo "Namespace 删除后仍存在: $$namespace" >&2; exit 1; fi; \
	echo "Namespace 已删除并确认不存在: $$namespace"

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
