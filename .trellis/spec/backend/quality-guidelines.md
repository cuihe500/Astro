# 质量规范

> 代码标准、校验要求与禁止模式。设计原则：KISS / YAGNI / 小步快跑（详见根目录 AGENTS.md）。

---

## 命令（只用 Makefile 目标，不手拼命令）

```bash
make build     # 编译到 bin/astro
make run       # 本地运行
make test      # go test -v ./...
make lint      # golangci-lint run
make swagger   # 重新生成 docs/（改了 handler 注解后必须跑）
make legacy-inventory # 仅在 test 环境只读盘点旧 App 与 Namespace
make clean
```

提交前最低要求：`make build` + `make lint` 通过。

## 代码标准

- `gofmt` 格式化；导出函数/类型必须有中文注释（`// CreateApp 创建应用`）。
- 注释、文档、提交信息统一中文；提交信息用语义化前缀（feat/fix/docs/style/refactor/test/chore）。
- 每个函数只做一件事；禁止无意义缩写命名。

## 分层强制要求

- **参数校验在 handler 层**：请求结构体用 `binding` tag（`binding:"required,min=0,max=10"`），`ShouldBindJSON` 失败即 `BadRequest` 返回。
- **权限检查在 service 层**：项目先校验 `Project.UserID`，应用再按 `project_id + app_id` 查询；Kubernetes 操作只使用已授权 Project 的 Namespace：

```go
if project.UserID != userID {
    return errcode.New(errcode.ErrForbidden)
}
```

- handler 从 `c.GetUint("user_id")` 取当前用户（auth 中间件注入），为 0 视为未登录。

## Swagger

每个 handler 方法带 swag 注解（`@Summary` / `@Tags` / `@Security Bearer` / `@Router`），格式参照 `internal/handler/app.go`。改注解后跑 `make swagger`。

## 测试现状
当前已有 OAuth2、项目和应用 Service 的最小单元测试，以及项目嵌套路由测试。新增复杂业务逻辑（service 层分支/权限判断）应补最小回归测试，但不强制补历史欠账。有测试后用 `make test` 跑。

## 场景：项目级资源归属

### 1. Scope / Trigger

- 新增或修改 Project API、项目内 App API、Project/App 数据模型或 Web 项目路由时适用。
- 项目是用户、应用与 Kubernetes Namespace 之间唯一的归属边界；不得恢复用户级 Namespace 或扁平 App 路由。

### 2. Signatures

- Project API：`POST/GET /api/v1/projects`、`GET/DELETE /api/v1/projects/:project_id`。
- App API：`/api/v1/projects/:project_id/apps` 及其 `/:id`、`/start`、`/stop`、`/restart`、`/logs` 子路由。
- Service：所有 App 操作同时接收 `projectID` 与 `userID`；创建请求还必须包含 `ProjectID`。
- Web：项目入口固定为 `/projects`，应用页面使用 `/projects/:projectId/apps...`。

### 3. Contracts

- handler 校验 `project_id`、`id` 为正整数，并校验请求体；service 校验 `Project.UserID == userID`。
- App 必须以 `project_id + app_id` 查询，Kubernetes 操作只使用已授权 Project 保存的 Namespace。
- 同一用户内 Project 名唯一；同一 Project 内 App 名唯一；App 的 `project_id` 非空且受外键约束。
- App 创建在锁定 Project 的事务内写入未提交记录并创建 Kubernetes 资源；只有回调和事务提交都成功后记录才可见，提交失败必须幂等删除已创建资源。
- 所有响应继续使用 `code`、`message`、`data`；旧 `/api/v1/apps` 不提供兼容入口。

### 4. Validation & Error Matrix

| 条件 | 结果 |
|---|---|
| `project_id` / `id` 非正整数 | `ErrBadRequest` |
| Project 不存在 | `ErrProjectNotFound` |
| Project 属于其他用户 | `ErrForbidden` |
| 用户内 Project 重名 | `ErrProjectExists` |
| Project 仍含 App | `ErrProjectNotEmpty`，不得删除 Namespace 或记录 |
| Project 内 App 重名 | `ErrAppExists` |
| K8s 创建失败 | 回滚未提交 App，返回应用创建错误 |
| K8s 创建成功但事务提交失败 | 删除 K8s 资源并返回错误；补偿失败时记录结构化错误并返回两次失败 |

### 5. Good / Base / Bad Cases

- Good：用户通过项目嵌套路由创建 App，数据库保存非空 `project_id`，资源位于该 Project 的 Namespace。
- Base：无 Project 用户只看到创建项目入口，不能调用扁平 App API。
- Bad：仅按 App ID 查询后再校验用户，或从客户端/App 字段接受 Namespace，可能造成跨项目访问或资源漂移。

### 6. Tests Required

- 路由测试断言全部 App 路由含 `:project_id`，旧 `/api/v1/apps` 返回不存在。
- Service 测试覆盖项目越权、错项目 App、项目内重名、非空项目删除和 K8s/数据库补偿失败。
- Repository/MariaDB 集成测试断言 Project 行锁使 App 创建与项目删除串行，未提交 App 对其他事务不可见。
- Web 测试断言项目 ID 在 API 与页面路由间完整传播，切换项目时不保留上一项目数据。

### 7. Wrong vs Correct

```go
// Wrong：只按 App ID 查询，Namespace 归属无法由路径中的 Project 约束。
app, err := repo.GetByID(appID)

// Correct：先确认 Project 所有权，再在同一 Project 内查询 App。
project, err := projects.GetByID(projectID)
app, err := repo.GetByProjectAndID(project.ID, appID)
```

## 场景：GitHub Actions Shell 静态检查一致性

### 1. Scope / Trigger

修改 `.github/workflows/*.yml` 中的多行 Shell，尤其是 `ssh` 远端命令时适用。`actionlint` 只有在执行环境存在 `shellcheck` 时才会报告 ShellCheck 规则，本地无报错不等于 CI 无报错。

### 2. Signatures

- 本地入口：`make workflow-lint`
- CI 入口：workflow 的“工作流静态检查”步骤

### 3. Contracts

- 未标注的 ShellCheck 告警必须修复。
- 确需客户端展开并传入受限远端命令的变量，只在对应命令前定点使用 `# shellcheck disable=SC2029`。
- 禁止文件级关闭规则或放宽 SSH、digest、host key 校验。

### 4. Validation & Error Matrix

| 情况 | 结果 |
|---|---|
| `actionlint` 或 CI ShellCheck 报错 | 阻止发布 |
| `SC2029` 且变量应在远端展开 | 修正引用方式，不抑制规则 |
| `SC2029` 且 digest 已在客户端校验、必须传给 forced command | 对单条 `ssh` 定点抑制 |
| 本地缺少 `shellcheck` | 只能声明基础 actionlint 通过，最终以 CI 结果为准 |

### 5. Good / Base / Bad Cases

- Good：digest 经正则校验，单条 `ssh` 前定点抑制 `SC2029`，CI 通过。
- Base：无嵌入式 Shell 的 workflow 直接通过 `make workflow-lint`。
- Bad：全局禁用 ShellCheck，或为消除告警而关闭严格 host key / 参数校验。

### 6. Tests Required

- 修改 workflow 后运行 `make workflow-lint`。
- 首次真实 push 必须确认 CI 中“工作流静态检查”通过后，镜像发布与部署 job 才能继续。

### 7. Wrong vs Correct

```yaml
# Wrong：范围过大，掩盖同一脚本中的其他 SC2029
# shellcheck disable=SC2029
run: |
  ssh "$host" deploy test "$api_reference" "$web_reference"

# Correct：仅标注已验证、预期在客户端展开的调用
run: |
  # shellcheck disable=SC2029
  ssh "$host" deploy test "$api_reference" "$web_reference"
```

## 禁止模式

- 一个实现就建 interface、为“将来扩展”留抽象层（YAGNI）。例外：跨系统边界的 `k8s.AppAdapter` 是既有约定。
- 用 `_` 忽略错误（见 error-handling.md）。
- 硬编码错误码数字、拼接 SQL、标准库 `log`。
- 在日志/响应中泄露密码与 Token。
