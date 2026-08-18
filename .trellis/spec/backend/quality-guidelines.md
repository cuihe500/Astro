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

**当前代码库没有任何 `_test.go` 文件**（技术债，已知现状）。新增复杂业务逻辑（service 层分支/权限判断）建议补最小化表驱动测试，但不强制补历史欠账。有测试后用 `make test` 跑。

## 禁止模式

- 一个实现就建 interface、为“将来扩展”留抽象层（YAGNI）。例外：跨系统边界的 `k8s.AppAdapter` 是既有约定。
- 用 `_` 忽略错误（见 error-handling.md）。
- 硬编码错误码数字、拼接 SQL、标准库 `log`。
- 在日志/响应中泄露密码与 Token。
