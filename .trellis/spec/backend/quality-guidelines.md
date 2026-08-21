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
make clean
```

提交前最低要求：`make build` + `make lint` 通过。

## 代码标准

- `gofmt` 格式化；导出函数/类型必须有中文注释（`// CreateApp 创建应用`）。
- 注释、文档、提交信息统一中文；提交信息用语义化前缀（feat/fix/docs/style/refactor/test/chore）。
- 每个函数只做一件事；禁止无意义缩写命名。

## 分层强制要求

- **参数校验在 handler 层**：请求结构体用 `binding` tag（`binding:"required,min=0,max=10"`），`ShouldBindJSON` 失败即 `BadRequest` 返回。
- **权限检查在 service 层**：所有资源操作必须校验归属，模式见 `internal/service/app.go`：

```go
if app.UserID != userID {
    return errcode.New(errcode.ErrForbidden)
}
```

- handler 从 `c.GetUint("user_id")` 取当前用户（auth 中间件注入），为 0 视为未登录。

## Swagger

每个 handler 方法带 swag 注解（`@Summary` / `@Tags` / `@Security Bearer` / `@Router`），格式参照 `internal/handler/app.go`。改注解后跑 `make swagger`。

## 测试现状
**当前代码库已有 `internal/service/oauth2_test.go`，覆盖 OAuth2 state 生成与校验；其他业务层仍缺少测试**（技术债）。新增复杂业务逻辑（service 层分支/权限判断）建议补最小化表驱动测试，但不强制补历史欠账。有测试后用 `make test` 跑。

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
